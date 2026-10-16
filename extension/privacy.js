/* Shared, dependency-free capture policy. Loaded by the extension and Node tests. */
const TyanPrivacy = (() => {
  const localHost = (h) => h === 'localhost' || h === '[::1]' || /^127\./.test(h);
  function server(value) {
    const u = new URL(value);
    if (!['http:', 'https:'].includes(u.protocol) || !localHost(u.hostname) || u.username || u.password || u.search || u.hash || (u.pathname !== '/' && u.pathname !== '')) throw new Error('Use a loopback TYAN URL, e.g. http://127.0.0.1:8787');
    return u.origin;
  }
  function labOrigin(value) {
    const u = new URL(value);
    const h = u.hostname;
    const privateHost = /^(10\.|192\.168\.|172\.(1[6-9]|2[0-9]|3[01])\.)/.test(h);
    if (!['http:', 'https:'].includes(u.protocol) || u.username || u.password || !(localHost(h) || privateHost || h.endsWith('.test') || h.endsWith('.localhost'))) throw new Error('Capture is limited to local lab origins (loopback, private IP, .test or .localhost).');
    return u.origin;
  }
  function safeUrl(value) {
    const u = new URL(value); u.username = ''; u.password = ''; u.hash = '';
    for (const k of [...u.searchParams.keys()]) u.searchParams.set(k, '[REDACTED]');
    return u.toString();
  }
  function headers(input) {
    const allowed = new Set(['content-type', 'content-length', 'accept', 'cache-control']);
    return Object.fromEntries(Object.entries(input || {}).filter(([k]) => allowed.has(k.toLowerCase())));
  }
  return { server, labOrigin, safeUrl, headers };
})();
if (typeof module !== 'undefined') module.exports = TyanPrivacy;
