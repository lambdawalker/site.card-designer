<script>
 import {EditorState,Compartment,Transaction} from '@codemirror/state';
 import {oneDark} from '@codemirror/theme-one-dark';
 import {onMount} from 'svelte';
 import {EditorView,basicSetup} from 'codemirror';
 import {html} from '@codemirror/lang-html';
 import {json} from '@codemirror/lang-json';
 import {css} from '@codemirror/lang-css';
 import {yaml} from '@codemirror/lang-yaml';
 import {StreamLanguage} from '@codemirror/language';
 import {formatSource} from './format-output.js';
 export let value='', language='html', change=()=>{}, label='',dark=false, readOnly=false;
 const yolo=StreamLanguage.define({
   token(stream){
     if(stream.eatSpace())return null;
     if(stream.match(/\d+(?:\.\d+)?(?=\s|$)/))return 'number';
     stream.skipToEnd();
     return 'string';
   }
 });
 const languages={html,css,json,yaml,yolo:()=>yolo};
 const colors=new Compartment(),wrapping=new Compartment();
 let wordWrap=localStorage.getItem('wrap:'+label)==='true';
 function toggleWrap(){wordWrap=!wordWrap;localStorage.setItem('wrap:'+label,String(wordWrap))}
 let host,view,formatting=false,formatError='';
 export async function format(){
   if(!view||readOnly||formatting)return;
   const editor=view,source=editor.state.doc.toString();
   formatting=true;formatError='';
   try{
     const formatted=await formatSource(source,language);
     // An edit or unmount while Prettier loads must not be overwritten.
     if(view!==editor||editor.state.doc.toString()!==source)return;
     if(formatted!==source){
       const {anchor,head}=editor.state.selection.main;
       editor.dispatch({changes:{from:0,to:source.length,insert:formatted},
         selection:{anchor:Math.min(anchor,formatted.length),head:Math.min(head,formatted.length)},
         annotations:Transaction.userEvent.of('format')});
     }
     editor.focus();
   }catch(e){if(view===editor&&editor.state.doc.toString()===source)formatError='Could not format code: '+e.message}
   finally{formatting=false}
 }
 function formatKey(event){
   if(!readOnly&&event.ctrlKey&&event.altKey&&!event.metaKey&&(event.code==='KeyL'||event.key.toLowerCase()==='l')){
     event.preventDefault();format();return true;
   }
   return false;
 }
 onMount(()=>{
   view=new EditorView({parent:host,doc:value,extensions:[EditorView.domEventHandlers({keydown:formatKey}),colors.of(dark?oneDark:[]),wrapping.of(wordWrap?EditorView.lineWrapping:[]),basicSetup,(languages[language]??html)(),EditorState.readOnly.of(readOnly),EditorView.editable.of(!readOnly),EditorView.contentAttributes.of({'aria-label':label,...(readOnly?{'role':'textbox','aria-readonly':'true',tabindex:'0'}:{})}),EditorView.updateListener.of(u=>{if(u.docChanged) change(u.state.doc.toString())})]});
   return ()=>{view.destroy();view=null};
 });
 $: if(view)view.dispatch({effects:wrapping.reconfigure(wordWrap?EditorView.lineWrapping:[])});
 $: if(view)view.dispatch({effects:colors.reconfigure(dark?oneDark:[])});
 $: if(view && value!==view.state.doc.toString()) view.dispatch({changes:{from:0,to:view.state.doc.length,insert:value}});
</script>
<div class="editor-wrapbar"><button type="button" aria-label={'Word wrap '+label} aria-pressed={wordWrap} on:click={toggleWrap}>Wrap</button></div>
<div class="editor" bind:this={host}></div>
{#if formatError}<p class="format-error" role="alert">{formatError}</p>{/if}
