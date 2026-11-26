<script lang="ts">
  import type { SearchView } from "../../../bindings/omni/internal/settingsui";

  interface Props {
    search: SearchView;
    /** Why the last attempt to start the download was refused, or "". */
    error: string;
    onenable: () => void;
    onskip: () => void;
    /** Pick another folder for the engine and the index, or go back to the usual one. */
    onchoose: () => void;
    onreset: () => void;
  }
  let { search, error, onenable, onskip, onchoose, onreset }: Props = $props();

  const gb = $derived((search.DownloadMB / 1024).toFixed(1));
</script>

<p class="eg">Finds “the clause about pets in the lease” as a PDF page, “retry with backoff” as a function, “dog on a beach” as a photo.</p>

<ul class="costs">
  <li>Downloads about {gb} GB once, then works offline and sends nothing anywhere.</li>
  <li>
    Saved in <span class="mono where">{search.Data.Engine}</span>; the index in <span class="mono where">{search.Data.Index}</span>.
    {#if search.Data.CanChange}
      <button type="button" class="btn quiet" onclick={onchoose}>Change folder...</button>
      {#if search.Data.Chosen}<button type="button" class="btn quiet" onclick={onreset}>Use the usual place</button>{/if}
    {/if}
  </li>
  <li>Uses your graphics card when it can. Nothing is indexed until you start it.</li>
</ul>
<p class="help mono">From {(search.Hosts ?? []).join(", ")}</p>

{#if search.Error}<p class="error">The last attempt failed: {search.Error}</p>{/if}
{#if error}<p class="error">{error}</p>{/if}

<div class="actions">
  <button type="button" class="btn primary" onclick={onenable}>{search.Installed ? "Turn on" : "Download and turn on"}</button>
  <button type="button" class="btn quiet" onclick={onskip}>Not now</button>
</div>

<style>
  .eg {
    margin: 0;
    color: var(--text-2);
    font-size: 12.5px;
    line-height: 1.5;
  }
  .costs {
    display: flex;
    flex-direction: column;
    gap: 5px;
    margin: 0;
    padding-left: 16px;
    color: var(--text-2);
    font-size: 12.5px;
    line-height: 1.5;
  }
  .where {
    overflow-wrap: anywhere;
    font-size: 11.5px;
  }
  .costs .btn {
    margin-left: 2px;
  }
  .actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }
</style>
