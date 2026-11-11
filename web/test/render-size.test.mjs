import {test} from 'node:test';
import assert from 'node:assert/strict';
import {renderSize} from '../render-size.js';
test('background backing size accounts for card dimensions, zoom and display density',()=>{
 assert.deepEqual(renderSize({width:1350,height:806},{scale:1.5,pixelRatio:2}),{width:4050,height:2418});
 assert.deepEqual(renderSize({width:740,height:1285},{scale:.5,pixelRatio:1}),{width:740,height:1285});
 assert.deepEqual(renderSize({}),{width:640,height:400});
});
test('background backing resolution stays within canvas limits and rejects invalid sizes',()=>{
 assert.deepEqual(renderSize({width:4096,height:2048},{scale:2,pixelRatio:3}),{width:4096,height:2048});
 assert.throws(()=>renderSize({width:0,height:400}));
 assert.throws(()=>renderSize({},{scale:NaN}));
});
