import assert from 'node:assert/strict';
import test from 'node:test';
import {formatSource} from '../format-output.js';

test('formats HTML while preserving many Jinja expressions, blocks and comments', async()=>{
  const tokens=Array.from({length:15},(_,i)=>`{{ person.field_${i} }}`);
  const source='<div>{% if person %}'+tokens.map(token=>`<p>${token}</p>`).join('')+'{# exact comment #}{% endif %}</div>';
  const output=await formatSource(source,'html');
  assert.ok(output.includes('\n'));
  for(const token of [...tokens,'{% if person %}','{% endif %}','{# exact comment #}'])assert.ok(output.includes(token),token);
  assert.equal(await formatSource(output,'html'),output);
});

test('formats CSS containing Jinja values and preserves quoted delimiters exactly', async()=>{
  const token='{{ "first }}   second" }}';
  const output=await formatSource(`a{content:${token};margin:0}`,'css');
  assert.ok(output.includes('margin: 0;'));
  assert.ok(output.includes(token));
  const html=await formatSource(`<p>${token}</p>`,'html');
  assert.ok(html.includes(token));
  assert.equal(await formatSource(output,'css'),output);
});

test('preserves CSS expression adjacency to units and selector/property names', async()=>{
  const output=await formatSource('.{{ klass }}{width:{{ width }}px;{{ property }}:{{ value }}%}','css');
  assert.ok(output.includes('.{{ klass }}'));
  assert.ok(output.includes('width: {{ width }}px;'));
  assert.ok(output.includes('{{ property }}: {{ value }}%;'));
});

test('reports invalid code instead of returning a partial formatting result',async()=>{
  await assert.rejects(()=>formatSource('div { color:','css'));
});
