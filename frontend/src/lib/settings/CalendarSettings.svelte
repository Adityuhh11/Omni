<script lang="ts">
  import { onMount } from "svelte";
  import { Calendar } from "../../../bindings/omni/internal/settingsui";
  import type { CalendarFeedView } from "../../../bindings/omni/internal/settingsui";
  import { colorOf } from "../calendar";

  let feeds = $state<CalendarFeedView[]>([]);
  let name = $state("");
  let address = $state("");
  let adding = $state(false);
  let refreshing = $state(false);
  let confirming = $state("");
  let error = $state("");
  let help = $state(false);

  const refresh = async () => {
    feeds = (await Calendar.View()) ?? [];
  };
  onMount(() => {
    refresh();
    const timer = setInterval(refresh, 5000);
    return () => clearInterval(timer);
  });

  async function add() {
    adding = true;
    error = await Calendar.Add(name, address);
    adding = false;
    if (error === "") {
      name = "";
      address = "";
    }
    await refresh();
  }

  async function remove(id: string) {
    confirming = "";
    error = await Calendar.Remove(id);
    await refresh();
  }

  async function syncNow() {
    refreshing = true;
    await Calendar.Refresh();
    await refresh();
    refreshing = false;
  }

  function ago(iso: string): string {
    if (!iso) return "Not fetched yet";
    const minutes = Math.round((Date.now() - new Date(iso).getTime()) / 60000);
    return minutes < 1 ? "Updated just now" : minutes < 60 ? `Updated ${minutes} min ago` : `Updated ${new Date(iso).toLocaleString(undefined, { hour: "numeric", minute: "2-digit", day: "numeric", month: "short" })}`;
  }
</script>

<div class="head-row">
  <h2>Calendar</h2>
  <button type="button" class="btn" onclick={() => Calendar.OpenCalendar()}>Open the calendar</button>
</div>

<h3 class="group-title">Calendars</h3>
<div class="group">
  {#each feeds as feed, i (feed.ID)}
    <div class="item">
      <span class="who">
        <span class="dot" style:background={colorOf(i)} aria-hidden="true"></span>
        <span class="text">
          <span class="name">{feed.Name}</span>
          {#if feed.Error}<span class="sub problem">{feed.Error}</span>
          {:else}<span class="sub">{ago(feed.Synced)}{feed.Synced ? ` · ${feed.Events} events` : ""}</span>{/if}
        </span>
      </span>
      <span class="end">
        {#if confirming === feed.ID}
          <button type="button" class="btn danger" onclick={() => remove(feed.ID)}>Remove</button>
          <button type="button" class="btn quiet" onclick={() => (confirming = "")}>Keep</button>
        {:else}
          <button type="button" class="btn quiet" onclick={() => (confirming = feed.ID)}>Remove</button>
        {/if}
      </span>
    </div>
  {:else}
    <p class="item sub">None yet. Add one below.</p>
  {/each}
  {#if feeds.length}
    <div class="item">
      <span class="sub">Fetched every 15 minutes while Omni runs</span>
      <button type="button" class="btn" disabled={refreshing} onclick={syncNow}>{refreshing ? "Refreshing…" : "Refresh now"}</button>
    </div>
  {/if}
</div>

<h3 class="group-title">Add a calendar</h3>
<form class="group" onsubmit={(e) => { e.preventDefault(); add(); }}>
  <div class="item">
    <label class="name" for="cal-name">Name</label>
    <input id="cal-name" type="text" placeholder="Work" bind:value={name} autocomplete="off" />
  </div>
  <div class="item">
    <span class="text">
      <label class="name" for="cal-address">Address</label>
      <span class="sub">Kept in the system credential store</span>
    </span>
    <input id="cal-address" type="password" class="mono" placeholder="https://… or webcal://…" bind:value={address} autocomplete="off" spellcheck="false" />
  </div>
  <div class="item">
    <button type="button" class="btn quiet where" onclick={() => (help = !help)} aria-expanded={help}>Where do I find it?</button>
    <button type="submit" class="btn primary" disabled={adding || !name.trim() || !address.trim()}>{adding ? "Adding…" : "Add"}</button>
  </div>
  {#if help}
    <div class="item stack how">
      <p><b>Google</b>: Calendar settings, the calendar under "Settings for my calendars", Integrate calendar, Secret address in iCal format.</p>
      <p><b>iCloud</b>: in the Calendar app, the share icon next to the calendar, Public Calendar, Copy Link.</p>
      <p><b>Outlook</b>: Settings, Calendar, Shared calendars, Publish a calendar, the ICS link.</p>
    </div>
  {/if}
</form>
{#if error}<p class="error">{error}</p>{/if}
<p class="group-note">Read only: Omni shows your calendars and changes nothing. Anyone with an address can read that calendar, so it never goes in a file.</p>

<style>
  .who {
    display: flex;
    align-items: center;
    gap: 12px;
    min-width: 0;
  }
  .dot {
    flex: none;
    width: 10px;
    height: 10px;
    border-radius: 50%;
  }
  .sub.problem {
    color: var(--danger);
  }
  .item.sub {
    min-height: 0;
    font-size: 12px;
  }
  .where {
    margin-left: -14px; /* the quiet button's padding, so its words line up with the rows' */
  }
  .how p {
    margin: 0;
    color: var(--text-2);
    font-size: 12.5px;
    line-height: 1.5;
  }
  .how b {
    color: var(--text);
    font-weight: 580;
  }
</style>
