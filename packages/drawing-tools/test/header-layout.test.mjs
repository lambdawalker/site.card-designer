import test from 'node:test';
import assert from 'node:assert/strict';
import {fitHeader} from '../src/draw/header/layout.ts';
import {parameters} from '../src/draw/card_background/parameters.ts';
const bounds={left:30,top:25,right:1320,bottom:152};
function recipe(component,depth=40){const r=parameters({family:'type-a',seed:42,color:'#aabbcc',width:1350,height:850});r.layers=[{role:'header',component:'header/'+component,parameters:{color:'#aabbcc',height:.25,depth,frequency:2}}];return r}
test('header shape covers measured content with padding at its shallowest edge',()=>{
 for(const [shape,depth,reserve] of [['square_header',40,0],['curved_header',40,0],['curved_header',-40,30],['sin_header',40,40]]){
  const input=recipe(shape,depth),output=fitHeader(input,bounds,12);
  assert.equal(output.layers[0].parameters.height,(152+12+reserve)/850);
  assert.deepEqual(output.header_layout,{version:1,bounds,padding:12});
  assert.equal(input.layers[0].parameters.height,.25);
  assert.equal(input.header_layout,undefined);
 }
});
test('changed text replaces saved measurements and preserves other layer parameters',()=>{
 const fitted=fitHeader(recipe('sin_header'),bounds,12);
 const changed=fitHeader(fitted,{...bounds,bottom:220},12);
 assert.equal(changed.layers[0].parameters.height,272/850);
 assert.equal(changed.layers[0].parameters.color,fitted.layers[0].parameters.color);
 assert.equal(changed.layers[0].parameters.depth,40);
 assert.deepEqual(fitHeader(recipe('square_header'),null).layers,recipe('square_header').layers);
});
test('rejects invalid measurement geometry instead of hiding header text',()=>{
 for(const bad of [{...bounds,bottom:NaN},{...bounds,left:-1},{...bounds,bottom:900},{...bounds,right:5}])assert.throws(()=>fitHeader(recipe('sin_header'),bad));
 assert.throws(()=>fitHeader(recipe('sin_header'),{...bounds,bottom:830}));
});
