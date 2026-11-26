<script lang="ts">
  import type { SearchView } from "../../../bindings/omni/internal/settingsui";
  import Folders from "./Folders.svelte";
  import Options from "./Options.svelte";

  interface Props {
    search: SearchView;
    /** Reads the settings and the search state again; the options only need the search state. */
    refresh: () => Promise<void>;
    refreshSearch: () => Promise<void>;
  }
  let { search, refresh, refreshSearch }: Props = $props();
</script>

<Folders {search} {refresh} />

<!-- The choices are made for this computer already (Setup.svelte); the person who wants to look can. -->
<details>
  <summary>How thorough? <span>Chosen for this computer</span></summary>
  <div class="body"><Options {search} refresh={refreshSearch} /></div>
</details>

<style>
  details {
    margin-top: 22px;
    border-radius: 14px;
    background: var(--tile);
    box-shadow: inset 0 0 0 1px var(--line);
  }
  summary {
    display: flex;
    align-items: baseline;
    gap: 10px;
    padding: 13px 16px;
    font: 600 13.5px/1.2 var(--sans);
    cursor: pointer;
  }
  summary span {
    color: var(--text-3);
    font-weight: 450;
    font-size: 12.5px;
  }
  .body {
    padding: 4px 16px 6px;
  }
</style>
