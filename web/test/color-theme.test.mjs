import test from 'node:test';
import assert from 'node:assert/strict';
import {themedDesign} from '../color-theme.js';

test('manual saved designs preserve their chosen colors',()=>{
 const design={seed:42,color:'#123456',text_color:'#ffffff'};
 assert.equal(themedDesign(design),design);
});
test('auto themes resolve from the seed consistently without changing the input',()=>{
 const design={auto_theme:true,seed:42,color:'#123456',text_color:'#ffffff'};
 const resolved=themedDesign(design);
 assert.notEqual(resolved.color,design.color);
 assert.equal(resolved.text_color,'#000000');
 assert.deepEqual(themedDesign(design),resolved);
 assert.equal(design.color,'#123456');
 assert.notEqual(themedDesign({...design,seed:43}).color,resolved.color);
});
