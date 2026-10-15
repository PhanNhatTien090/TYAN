import assert from "node:assert/strict";
import test from "node:test";
import { readFileSync } from "node:fs";
import { activeMention, mentionKinds, mentionSearch, mentionToken, selectedMentions } from "./chat-mentions.ts";

test("mention trigger supports Chinese and cursor placement without hijacking email", () => {
  assert.equal(activeMention("user@example.com", 16), null);
  assert.equal(activeMention("已选 @[漏洞#1 X]", 12), null);
  assert.deepEqual(activeMention("查看@漏洞 后面的文字", 5), { start: 2, end: 5, query: "漏洞" });
  assert.equal(activeMention("@漏洞\n下一行", 8), null);
});

test("categories, Chinese aliases, IP and keyword search", () => {
  assert.equal(mentionSearch("").categories.length, 9);
  assert.equal(mentionSearch("漏").categories[0].kind, "finding");
  assert.equal(mentionSearch("漏洞").kind, "finding");
  assert.equal(mentionSearch("漏洞SQL注入").query, "SQL注入");
  assert.equal(mentionSearch("ip 192.0.2.1").kind, "ip");
  assert.equal(mentionSearch("接口 GET /api").query, "GET /api");
  assert.equal(mentionSearch("acme.com").kind, "");
});

test("tokens roundtrip labels and removing one reference preserves its neighbors", () => {
  const first = mentionToken({ kind: "finding", id: 12, label: "标题[1]\n描述" });
  const second = mentionToken({ kind: "ip", id: 13, label: "192.0.2.1" });
  const value = `分析 ${first} 和 ${second}`;
  const selected = selectedMentions(value);
  assert.equal(selected.length, 2);
  assert.equal(selected[0].label, "Finding #12 · 标题(1) 描述");
  const next = value.slice(0, selected[0].start) + value.slice(selected[0].start + selected[0].token.length);
  assert.equal(selectedMentions(next)[0].token, second);
});

test("English references and legacy history remain readable", () => {
  assert.equal(mentionSearch("finding SQL injection").kind, "finding");
  assert.equal(mentionSearch("organization Acme").kind, "company");
  assert.equal(selectedMentions("@[Finding#12 SQL injection] @[漏洞#13 Existing finding]").length, 2);
});

test("every visible reference kind is accepted by the backend wire pattern", () => {
  const source = readFileSync(new URL("../../../server/chat_mentions.go", import.meta.url), "utf8");
  const expression = source.match(/chatMentionPattern = regexp\.MustCompile\(`([^`]+)`\)/)?.[1];
  assert.ok(expression);
  const pattern = new RegExp(expression);
  for (const item of mentionKinds) {
    assert.ok(pattern.test(`@[${item.label}#12 Example]`), item.label);
    assert.ok(pattern.test(`@[${item.legacy}#12 Existing]`), item.legacy);
  }
});
