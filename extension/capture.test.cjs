const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const policy = require('./privacy.js');

test('background capture sends only sanitized metadata to loopback', async () => {
  const uploads = [];
  const local = {serverUrl:'http://127.0.0.1:8787',labOrigin:'http://127.0.0.1:3000'};
  const session = {token:'FAKE_OPERATOR_TOKEN'};
  const context = vm.createContext({
    TyanPrivacy:policy, importScripts:()=>{}, URL, Map,
    fetch:async(url,options)=>{uploads.push({url,options});return {ok:true,json:async()=>({ingested:1,failed:0})};},
    chrome:{
      storage:{local:{get:async()=>local},session:{get:async()=>session,set:async values=>Object.assign(session,values)}},
      tabs:{query:async()=>[{id:1}],get:async()=>({url:local.labOrigin+'/'}),onUpdated:{addListener:()=>{}}},
      debugger:{attach:async()=>{},detach:async()=>{},sendCommand:async()=>{},onEvent:{addListener:()=>{}},onDetach:{addListener:()=>{}}},
      action:{setBadgeText:()=>{},setBadgeBackgroundColor:()=>{}},runtime:{onMessage:{addListener:()=>{}}},
    },
  });
  vm.runInContext(fs.readFileSync(__dirname+'/background.js','utf8'),context);
  await vm.runInContext(`(async()=>{
    await start();
    await queueExchange({method:'POST',url:'http://127.0.0.1:3000/login?token=QUERY_SECRET',headers:{Authorization:'Bearer HEADER_SECRET',Cookie:'COOKIE_SECRET'},postData:'BODY_SECRET'}, {status:200,headers:{'set-cookie':'RESPONSE_SECRET'}});
    await queueExchange({method:'GET',url:'https://github.com/private',headers:{}},{status:200,headers:{}});
    await flush();
  })()`,context);
  assert.equal(uploads.length,1);
  assert.equal(uploads[0].url,'http://127.0.0.1:8787/api/traffic/ingest');
  assert.equal(uploads[0].options.redirect,'error');
  const body=uploads[0].options.body;
  assert(!/QUERY_SECRET|HEADER_SECRET|COOKIE_SECRET|BODY_SECRET|RESPONSE_SECRET|github\.com/.test(body));
  const exchanges=JSON.parse(body).exchanges;
  assert.equal(exchanges.length,1);
  assert.equal(exchanges[0].requestBody,'');
  assert.equal(exchanges[0].responseBody,'');
});
