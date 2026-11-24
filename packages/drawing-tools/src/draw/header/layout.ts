import type {Recipe} from '../card_background/parameters.ts';

export interface HeaderBounds {left:number;top:number;right:number;bottom:number}
export interface HeaderLayout {version:1;bounds:HeaderBounds;padding:number}

/** Fit the shallowest part of each header shape below already-laid-out content. */
export function fitHeader(recipe:Recipe,bounds:HeaderBounds|null,padding=12):Recipe {
 const result=structuredClone(recipe);
 delete result.header_layout;
 if(!bounds)return result;
 const {left,top,right,bottom}=bounds;
 if(![left,top,right,bottom,padding].every(Number.isFinite)||left<0||top<0||right<left||bottom<top||right>recipe.width||bottom>recipe.height||padding<0)
  throw new Error('Invalid rendered header bounds');
 let fitted=false;
 for(const layer of result.layers){
  if(layer.role!=='header')continue;
  const p=layer.parameters as unknown as {height:number;depth:number};
  let reserve=0;
  if(layer.component==='header/sin_header')reserve=Math.abs(p.depth);
  else if(layer.component==='header/curved_header')reserve=Math.max(0,-p.depth*.75);
  else if(layer.component!=='header/square_header')throw new Error('Unsupported header geometry');
  const height=bottom+padding+reserve;
  if(!Number.isFinite(height)||height>recipe.height)throw new Error('Rendered header does not fit inside the card');
  p.height=height/recipe.height;fitted=true;
 }
 if(fitted)result.header_layout={version:1,bounds:{...bounds},padding};
 return result;
}
