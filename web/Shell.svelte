<script>
import {onMount} from 'svelte';import Studio from './Studio.svelte';import {finishSignIn,signIn,signOut,authenticated} from './auth.js';import {request} from './api.js';
let ready=false,signed=false,error='',tokens=[],tokenName='',newToken='',showTokens=false;
onMount(async()=>{try{await finishSignIn();signed=authenticated()}catch(e){error=e.message}finally{ready=true}});
async function manage(){showTokens=!showTokens;if(showTokens)try{tokens=await request('tokens')}catch(e){error=e.message}}
async function create(){try{const result=await request('tokens','POST',{name:tokenName});newToken=result.token;tokenName='';tokens=await request('tokens')}catch(e){error=e.message}}
async function revoke(id){try{await request('tokens/'+encodeURIComponent(id),'DELETE');tokens=await request('tokens')}catch(e){error=e.message}}
</script>
<div class="account-bar"><span>Card Designer · Synthetic specimens</span><div>{#if signed}<button on:click={manage}>API tokens</button><button on:click={signOut}>Sign out</button>{:else}<button on:click={()=>signIn().catch(e=>error=e.message)}>Sign in</button>{/if}</div></div>
{#if error}<div role="alert" class="account-error">{error}<button on:click={()=>error=''}>Dismiss</button></div>{/if}
{#if showTokens}<section class="token-panel"><h2>Read-only API tokens</h2><p>Tokens expire after 90 days and can read your bundles and dataset packages.</p><form on:submit|preventDefault={create}><input aria-label="Token name" bind:value={tokenName} required maxlength="100"><button>Create token</button></form>{#if newToken}<p>Copy this token now. It is shown only once.</p><input aria-label="New API token" readonly value={newToken}><button on:click={()=>newToken=''}>Hide token</button>{/if}{#each tokens as token}<p>{token.name} · expires {new Date(typeof token.expires_at==='number'?token.expires_at*1000:token.expires_at).toLocaleDateString()}<button on:click={()=>revoke(token.id)}>Revoke</button></p>{/each}</section>{/if}
{#if ready}<Studio/>{/if}
<style>.account-bar{height:40px;background:#173e42;color:white;display:flex;align-items:center;justify-content:space-between;padding:0 18px;font:13px system-ui}.account-bar button{margin-left:8px}.account-error,.token-panel{padding:14px;background:#f8eed4;color:#222}.token-panel input{min-width:240px}.account-error button{margin-left:20px}:global(body){height:100vh}:global(#app){height:100%;display:flex;flex-direction:column}:global(.studio){flex:1;min-height:0}</style>
