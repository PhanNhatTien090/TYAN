importScripts("privacy.js");
// TYAN Traffic Capture — background service worker.
//
// Captures sanitized metadata from one explicitly configured local lab tab through the chrome.debugger API
// (Network domain) and pushes text request/response pairs to the TYAN
// ingest endpoint in batches. Bodies and credential headers are never uploaded. State lives in chrome.storage.session so the
// popup and the (short-lived) service worker share one view; the queue is
// deliberately in-memory — a browser restart drops unsent entries, which is
// the acceptable failure mode for a manual capture session.

const FLUSH_AT = 10; // entries per batch

let session = {
  capturing: false,
  tabId: null,
  queue: [], // pending ExternalExchange objects
  requests: new Map(), // debugger requestId -> partial request info
  ingested: 0,
  failed: 0,
  lastError: "",
};

async function saveState() {
  await chrome.storage.session.set({
    capturing: session.capturing,
    tabId: session.tabId,
    ingested: session.ingested,
    failed: session.failed,
    lastError: session.lastError,
    queued: session.queue.length,
  });
}

async function getConfig() {
  const { serverUrl = "", labOrigin = "" } = await chrome.storage.local.get(["serverUrl", "labOrigin"]);
  const { token = "" } = await chrome.storage.session.get(["token"]);
  return { serverUrl: serverUrl ? TyanPrivacy.server(serverUrl) : "", labOrigin: labOrigin ? TyanPrivacy.labOrigin(labOrigin) : "", token };
}

async function getActiveTabId() {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  return tab ? tab.id : null;
}

async function start() {
  if (session.capturing) return { ok: true };
  const { serverUrl, token, labOrigin } = await getConfig();
  if (!serverUrl || !token || !labOrigin) {
    return { ok: false, error: "Set the TYAN URL and token first." };
  }
  const tabId = await getActiveTabId();
  if (tabId == null) return { ok: false, error: "No active tab." };
  const tab = await chrome.tabs.get(tabId);
  if (new URL(tab.url).origin !== labOrigin) return { ok: false, error: "Active tab must match the configured lab origin." };
  try {
    await chrome.debugger.attach({ tabId }, "1.3");
  } catch (e) {
    return { ok: false, error: `attach failed: ${e.message}` };
  }
  await chrome.debugger.sendCommand({ tabId }, "Network.enable", {});
  session = { ...session, capturing: true, tabId, queue: [], requests: new Map(), ingested: 0, failed: 0, lastError: "" };
  chrome.action.setBadgeText({ text: "REC" });
  chrome.action.setBadgeBackgroundColor({ color: "#c2410c" });
  await saveState();
  return { ok: true };
}

async function stop() {
  if (session.capturing && session.tabId != null) {
    await flush();
    try {
      await chrome.debugger.sendCommand({ tabId: session.tabId }, "Network.disable", {});
      await chrome.debugger.detach({ tabId: session.tabId });
    } catch (_) {
      // tab may already be gone; nothing to do
    }
  }
  session.capturing = false;
  session.tabId = null;
  chrome.action.setBadgeText({ text: "" });
  await saveState();
  return { ok: true };
}

async function queueExchange(info, response) {
  const { labOrigin } = await getConfig();
  if (new URL(info.url).origin !== labOrigin) return;
  session.queue.push({
    method: info.method,
    url: TyanPrivacy.safeUrl(info.url),
    status: response.status || 0,
    requestHeaders: TyanPrivacy.headers(info.headers),
    requestBody: "",
    responseHeaders: TyanPrivacy.headers(response.headers),
    responseBody: "",
  });
  if (session.queue.length >= FLUSH_AT) await flush();
}

async function flush() {
  if (session.queue.length === 0) return;
  const { serverUrl, token, labOrigin } = await getConfig();
  if (!serverUrl || !token || !labOrigin) {
    session.lastError = "missing server URL or token";
    await saveState();
    return;
  }
  const batch = session.queue.splice(0, FLUSH_AT);
  try {
    const res = await fetch(`${serverUrl}/api/traffic/ingest`, {
      method: "POST",
      headers: { "Authorization": `Bearer ${token}`, "Content-Type": "application/json" },
      body: JSON.stringify({ exchanges: batch }),
      redirect: "error",
    });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const out = await res.json();
    session.ingested += out.ingested || 0;
    session.failed += out.failed || 0;
    session.lastError = out.failed ? `${out.failed} entries rejected by the server` : "";
  } catch (e) {
    // put the batch back (oldest first) but cap the queue so a dead server
    // cannot grow memory unbounded
    session.queue = batch.concat(session.queue).slice(0, 200);
    session.lastError = e.message;
  }
  await saveState();
}

chrome.debugger.onEvent.addListener((source, method, params) => {
  if (!session.capturing || !source.tabId || source.tabId !== session.tabId) return;
  if (method === "Network.requestWillBeSent") {
    session.requests.set(params.requestId, {
      requestId: params.requestId,
      method: params.request.method,
      url: params.request.url,
      headers: params.request.headers,
      postData: "",
    });
  } else if (method === "Network.responseReceived") {
    const info = session.requests.get(params.requestId);
    if (info) info.response = params.response;
  } else if (method === "Network.loadingFinished") {
    const info = session.requests.get(params.requestId);
    session.requests.delete(params.requestId);
    if (!info || !info.response) return;
    queueExchange(info, info.response).catch((e) => {
      session.lastError = e.message;
      saveState();
    });
  }
});

chrome.debugger.onDetach.addListener((source) => {
  if (session.capturing && source.tabId === session.tabId) {
    // DevTools opened, or the tab navigated somewhere undebaggable: stop
    // cleanly instead of silently dropping events.
    session.capturing = false;
    session.tabId = null;
    session.lastError = "debugger detached (DevTools opened?)";
    chrome.action.setBadgeText({ text: "" });
    saveState();
  }
});

chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
  (async () => {
    switch (msg?.type) {
      case "getState": {
        sendResponse({
          capturing: session.capturing,
          tabId: session.tabId,
          ingested: session.ingested,
          failed: session.failed,
          queued: session.queue.length,
          lastError: session.lastError,
          configured: !!(await getConfig()).serverUrl,
        });
        return;
      }
      case "start":
        sendResponse(await start());
        return;
      case "stop":
        sendResponse(await stop());
        return;
      case "flushNow":
        await flush();
        sendResponse(await stateSnapshot());
        return;
      default:
        sendResponse({ ok: false, error: "unknown message" });
    }
  })().catch(e => sendResponse({ ok: false, error: e.message }));
  return true; // async sendResponse
});

async function stateSnapshot() {
  return {
    capturing: session.capturing,
    tabId: session.tabId,
    ingested: session.ingested,
    failed: session.failed,
    queued: session.queue.length,
    lastError: session.lastError,
  };
}

chrome.tabs.onUpdated.addListener((tabId, change) => {
  if (tabId !== session.tabId || !change.url) return;
  getConfig().then(config => {
    if (new URL(change.url).origin !== config.labOrigin) return stop();
  }).catch(() => stop());
});
