const {test} = require('node:test');
const assert = require('node:assert/strict');
const policy = require('./privacy.js');
test('capture destination rejects remote hosts, credentials and path tricks', () => {
  assert.equal(policy.server('http://127.0.0.1:8787/'),'http://127.0.0.1:8787');
  for (const u of ['https://attacker.example','http://localhost.attacker.example','http://secret@localhost:8787','http://localhost:8787/redirect','file:///tmp']) assert.throws(()=>policy.server(u));
});
test('capture target must be an isolated lab origin', () => {
  assert.equal(policy.labOrigin('http://192.168.1.3:3000/a'),'http://192.168.1.3:3000');
  assert.throws(()=>policy.labOrigin('https://github.com'));
  assert.throws(()=>policy.labOrigin('https://linkedin.com'));
});
test('query credentials, fragments and sensitive headers are stripped', () => {
  const safe = policy.safeUrl('http://user:secret@localhost/a?token=abc&email=x%40y#private');
  assert(!safe.includes('secret')); assert(!safe.includes('abc')); assert(!safe.includes('x%40y')); assert(!safe.includes('private'));
  assert.deepEqual(policy.headers({Authorization:'Bearer secret',Cookie:'secret','Set-Cookie':'secret','Content-Type':'application/json','X-API-Key':'secret'}),{'Content-Type':'application/json'});
});
