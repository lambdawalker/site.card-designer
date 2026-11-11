import {measureHeader} from './header-measurement.js';
import {parameters, replay, fitHeader} from '@lambdawalker/drawing-tools';
import {renderSize} from './render-size.js';
import {themedDesign} from './color-theme.js';
import {api,selectedPerson} from './api.js';
import {renderTemplate} from './client-renderer.js';
export {api};
const preview=async (design,person)=>renderTemplate(design,person??await selectedPerson(design.sample,design.sample_source));
const generations=new Map();
function procedural(design,target=renderSize(design),bounds){
  const {seed,color}=design,width=design.width||640,height=design.height||400,family=design.background.replace('-v1','');
  const saved=design.background_recipe;
  const matching=saved && saved.family===family && saved.seed===seed && saved.color===color && saved.width===width && saved.height===height ? saved : null;
  const base=matching||parameters({family,seed,color,width,height});
  const recipe=bounds===undefined?base:fitHeader(base,bounds);
  const key=JSON.stringify([family,seed,color,width,height,recipe,target.width,target.height]);
  if(!generations.has(key)){
    const generation=(async()=>{
      const generated=await replay(recipe,target);
      return {recipe:generated.recipe,url:generated.canvas.toDataURL('image/png')};
    })();
    generations.set(key,generation);
    generation.catch(()=>generations.delete(key));
    if(generations.size>8)generations.delete(generations.keys().next().value);
  }
  return generations.get(key);
}
function cardHTML(design,result,background='transparent'){
  const width=design.width||640,height=design.height||400;
  const policy="default-src 'none'; style-src 'unsafe-inline'; font-src data:; img-src data:; base-uri 'none'; form-action 'none'";
  return '<!doctype html><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="'+policy+'"><style>:root{color-scheme:light;color:'+(design.text_color||'#000000')+'}*{box-sizing:border-box}body{margin:0}#view-port{position:relative;width:'+width+'px;height:'+height+'px;overflow:hidden;background:'+background+'}'+result.css+'</style><div id="view-port">'+result.template+'</div>';
}
function hasHeader(design){return ['type-a-v1','type-c-v1'].includes(design.background)||design.background_recipe?.layers?.some(layer=>layer.role==='header')}
async function headerBounds(design,result){
  return hasHeader(design)?measureHeader(cardHTML(design,result),design.width||640,design.height||400):undefined;
}
export async function backgroundRecipe(design){
  design=themedDesign(design);
  if(!design.background.endsWith('-v1'))return null;
  let bounds;
  if(hasHeader(design)){
    // Invalid editor source/layout may still be saved; its preview reports the error.
    try{const result=await preview(design);bounds=await headerBounds(design,result)}
    catch{return (await procedural(design)).recipe}
  }
  return (await procedural(design,renderSize(design),bounds)).recipe;
}
export async function renderCard(design,options={}) {
  design=themedDesign(design);
  const target=renderSize(design,{pixelRatio:window.devicePixelRatio||1,...options});
  const result = await preview(design,options.person);
  const {seed,color}=design,width=design.width||640,height=design.height||400;
  let background,background_recipe=null;
  if(design.background.startsWith('asset-')){
    const url=design.background_image;if(!url?.startsWith('data:image/'))throw Error('Missing embedded background image');
    const image=new Image();image.src=url;await image.decode();
    background='url('+url+') center/cover';
  }else if(design.background==='plain')background=color;
  else if(design.background==='type-b')background='url('+window.typeBBackground(width,height,color,seed,target)+') center/cover';
  else{
    const bounds=await headerBounds(design,result);
    const generated=await procedural(design,target,bounds);
    background_recipe=generated.recipe;
    background='url('+generated.url+') center/cover';
  }
  const html=cardHTML(design,result,background);
  return {html,...result,background_recipe,background_size:target};
}
window.cardDesignerRender=renderCard;
