<script lang="ts">
  import { onMount } from "svelte";
  import { Browser } from "@wailsio/runtime";
  import { Calendar } from "../../../bindings/omni/internal/settingsui";
  import type { CalendarFeedView } from "../../../bindings/omni/internal/settingsui";
  import { colorOf } from "../calendar";
  import { isMac } from "../platform";

  let feeds = $state<CalendarFeedView[]>([]);
  let name = $state("");
  let address = $state("");
  let adding = $state(false);
  let refreshing = $state(false);
  let confirming = $state("");
  let error = $state("");
  let help = $state(false);

  // Where each service keeps a calendar's subscription address. Google's page for one calendar needs its id, so its link
  // opens the settings and the steps say the rest.
  const providers = [
    {
      name: "Google Calendar",
      open: "https://calendar.google.com/calendar/r/settings",
      steps: "Pick the calendar under “Settings for my calendars”, scroll to “Integrate calendar”, copy “Secret address in iCal format”.",
    },
    {
      name: "Outlook.com",
      open: "https://outlook.live.com/calendar/0/options/calendar/SharedCalendars",
      steps: "Under “Publish a calendar”, choose the calendar and “Can view all details”, Publish, then copy the ICS link.",
    },
    {
      name: "Outlook for work or school",
      open: "https://outlook.office.com/calendar/options/calendar/SharedCalendars",
      steps: "The same as Outlook.com. If “Publish a calendar” is missing, your organization has switched it off.",
    },
    {
      name: "iCloud",
      open: "https://www.icloud.com/calendar/",
      steps: "Click the share icon next to the calendar, tick “Public Calendar”, Copy Link.",
    },
  ];

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
    try {
      error = await Calendar.Add(name, address);
      if (error === "") {
        name = "";
        address = "";
      }
    } finally {
      adding = false; // the button comes back even if the call failed
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
    try {
      await Calendar.Refresh();
      await refresh();
    } finally {
      refreshing = false;
    }
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
    <button type="button" class="btn quiet where" onclick={() => (help = !help)} aria-expanded={help} aria-controls="cal-help">
      Where do I find it?<span class="chev" class:open={help} aria-hidden="true"></span>
    </button>
    <button type="submit" class="btn primary" disabled={adding || !name.trim() || !address.trim()}>{adding ? "Adding…" : "Add"}</button>
  </div>
  {#if help}
    <div class="item stack how" id="cal-help">
      {#each providers as p (p.name)}
        <div class="provider">
          <span class="text">
            <span class="name">{p.name}</span>
            <span class="sub">{p.steps}</span>
          </span>
          <button type="button" class="btn" onclick={() => Browser.OpenURL(p.open)} title={p.open}>Open ↗</button>
        </div>
      {/each}
      <p class="sub">The address ends in <span class="mono">.ics</span> or starts with <span class="mono">webcal://</span>. Sign in first if the page asks.</p>
    </div>
  {/if}
</form>
{#if error}<p class="error">{error}</p>{/if}
<p class="group-note">Read only: Omni shows your calendars and changes nothing. Anyone with an address can read that calendar, so it never goes in a file.</p>
{#if isMac}
  <p class="group-note">Calendars in the Mac's Calendar app are not read by themselves yet: add each one here by its address.</p>
{/if}

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
    display: inline-flex;
    align-items: center;
    gap: 8px;
    margin-left: -14px; /* the quiet button's padding, so its words line up with the rows' */
  }
  .chev {
    width: 6px;
    height: 6px;
    margin-top: -3px;
    border-right: 1.5px solid currentColor;
    border-bottom: 1.5px solid currentColor;
    transform: rotate(45deg);
    transition: transform 0.15s var(--ease);
  }
  .chev.open {
    margin-top: 3px;
    transform: rotate(-135deg);
  }
  @media (prefers-reduced-motion: reduce) {
    .chev {
      transition: none;
    }
  }
  .how {
    gap: 14px;
  }
  .provider {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
  }
  .provider .btn {
    flex: none;
  }
  .how .sub {
    line-height: 1.45;
  }
  .how > p.sub {
    margin: 0;
    color: var(--text-3);
    font-size: 12px;
  }
</style>
