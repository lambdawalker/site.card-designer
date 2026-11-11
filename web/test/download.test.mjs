import test from 'node:test';
import assert from 'node:assert/strict';
import {zipFiles,downloadName} from '../download.js';

test('ZIP stores multiple PNGs with UTF-8 names and matching CRCs', async()=>{
 const bytes=new Uint8Array(await (await zipFiles([{name:'card-one.png',blob:new Blob(['first'])},{name:'cárd-two.png',blob:new Blob(['second'])}])).arrayBuffer());
 const view=new DataView(bytes.buffer);
 assert.equal(view.getUint32(0,true),0x04034b50);
 assert.equal(view.getUint16(6,true),0x800);
 assert.equal(view.getUint32(14,true),0x9271ee57);
 assert.equal(view.getUint32(bytes.length-22,true),0x06054b50);
 assert.equal(view.getUint16(bytes.length-12,true),2);
 assert.equal(new TextDecoder().decode(bytes).includes('cárd-two.png'),true);
});
test('Download names are portable with a fallback',()=>{
 assert.equal(downloadName('Driver / A: left'),'Driver-A-left');
 assert.equal(downloadName('???'),'card');
});
