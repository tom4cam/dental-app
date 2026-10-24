<script lang="ts">
  import type {
    Appointment,
    CountryConfig,
    Operatory,
    Patient,
    PatientAccount,
    Payment,
    Provider,
  } from "@bindings/domain/models.js";
  import { PaymentMethod } from "@bindings/domain/models.js";
  import { BillingService } from "@bindings/services/index.js";
  import { formatCurrency } from "$lib/currency.js";
  import { getLocalDateString, getTodayDateString, parseLocalDate } from "../lib/date.js";
  import { handleError } from "../lib/error.js";
  import { paymentMethodLabel } from "../lib/labels.js";
  import { m } from "../paraglide/messages.js";
  import { auth } from "../stores/auth.svelte.js";
  import EmptyState from "../components/ui/EmptyState.svelte";
  import StatusBadge from "../components/ui/StatusBadge.svelte";

  let {
    appointments = [],
    patients = [],
    providers = [],
    operatories = [],
    countryMeta = null,
    onupdatestatus,
  } = $props<{
    appointments?: Appointment[];
    patients?: Patient[];
    providers?: Provider[];
    operatories?: Operatory[];
    countryMeta?: CountryConfig | null;
    onupdatestatus: (id: string, status: string) => void;
  }>();

  // Desk payments only: insurance remittances and write-offs are recorded in Billing.
  const deskMethods = [
    PaymentMethod.PaymentMethodCash,
    PaymentMethod.PaymentMethodCheck,
    PaymentMethod.PaymentMethodCreditCard,
  ];

  let date = $state(getTodayDateString());
  let expandedId = $state("");

  let account = $state<PatientAccount | null>(null);
  let accountLoading = $state(false);
  let accountError = $state("");
  let accountGen = 0;

  let amount = $state("");
  let method = $state<string>(PaymentMethod.PaymentMethodCash);
  let saving = $state(false);
  let payMessage = $state("");
  let payError = $state("");

  let dayAppointments = $derived(
    appointments
      .filter((a: Appointment) => getLocalDateString(a.start_time) === date)
      .sort((a: Appointment, b: Appointment) => a.start_time.localeCompare(b.start_time))
  );

  let stats = $derived.by(() => {
    const live = dayAppointments.filter((a: Appointment) => a.status !== "cancelled");
    const count = (...statuses: string[]) =>
      live.filter((a: Appointment) => statuses.includes(a.status)).length;
    return {
      total: live.length,
      waiting: count("arrived", "in_chair"),
      upcoming: count("scheduled", "confirmed"),
      done: count("completed"),
    };
  });

  let dateHeading = $derived.by(() => {
    try {
      return parseLocalDate(date).toLocaleDateString([], {
        weekday: "long",
        month: "long",
        day: "numeric",
      });
    } catch {
      return date;
    }
  });

  let balanceDue = $derived(account ? Math.max(account.outstanding, 0) : 0);

  let accountRows = $derived(
    account
      ? [
          { label: m.frontdesk_billed(), cents: account.total_billed },
          { label: m.frontdesk_paid_patient(), cents: account.paid_by_patient },
          { label: m.frontdesk_paid_insurance(), cents: account.paid_by_insurance },
          { label: m.frontdesk_written_off(), cents: account.written_off },
        ]
      : []
  );

  function changeDay(delta: number) {
    const d = parseLocalDate(date);
    d.setDate(d.getDate() + delta);
    date = getLocalDateString(d);
  }

  function findPatient(id: string): Patient | undefined {
    return patients.find((p: Patient) => p.id === id);
  }

  function patientName(id: string): string {
    const p = findPatient(id);
    return p ? `${p.first_name} ${p.last_name}` : m.appts_unknown_patient();
  }

  function providerName(id: string): string {
    return providers.find((p: Provider) => p.id === id)?.name || "";
  }

  function operatoryName(id: string): string {
    return operatories.find((o: Operatory) => o.id === id)?.name || "";
  }

  function formatTime(iso: string): string {
    const d = new Date(iso);
    if (isNaN(d.getTime())) return iso;
    return d.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
  }

  function statusLabel(status: string): string {
    switch (status) {
      case "scheduled":
        return m.appts_status_scheduled();
      case "confirmed":
        return m.appts_status_confirmed();
      case "arrived":
        return m.appts_status_arrived();
      case "in_chair":
        return m.appts_status_in_chair();
      case "completed":
        return m.appts_status_completed();
      case "cancelled":
        return m.appts_status_cancelled();
      case "no_show":
        return m.appts_status_no_show();
      default:
        return status;
    }
  }

  function money(cents: number): string {
    return formatCurrency(cents, countryMeta?.default_currency);
  }

  async function loadAccount(patientId: string) {
    const gen = ++accountGen;
    account = null;
    accountError = "";
    accountLoading = true;
    try {
      const res = await BillingService.GetPatientAccount(auth.token, patientId);
      if (gen === accountGen) account = res;
    } catch (err) {
      if (gen === accountGen) accountError = handleError(err, m.frontdesk_account_error());
    } finally {
      if (gen === accountGen) accountLoading = false;
    }
  }

  function toggleDetails(appt: Appointment) {
    if (expandedId === appt.id) {
      expandedId = "";
      return;
    }
    expandedId = appt.id;
    amount = "";
    method = PaymentMethod.PaymentMethodCash;
    payMessage = "";
    payError = "";
    loadAccount(appt.patient_id);
  }

  async function recordPayment(e: Event, patientId: string) {
    e.preventDefault();
    payMessage = "";
    payError = "";
    const cents = Math.round(parseFloat(amount) * 100);
    if (isNaN(cents) || cents <= 0) {
      payError = m.frontdesk_amount_invalid();
      return;
    }
    const payment: Omit<Payment, "created_at"> = {
      id: `pay_${Date.now()}`,
      patient_id: patientId,
      claim_id: "",
      amount: cents,
      method: method as PaymentMethod,
      date: getTodayDateString(),
      notes: "",
    };
    saving = true;
    try {
      await BillingService.RecordPayment(auth.token, payment as unknown as Payment);
      amount = "";
      payMessage = m.frontdesk_payment_saved();
      await loadAccount(patientId);
    } catch (err) {
      payError = handleError(err, m.frontdesk_payment_error());
    } finally {
      saving = false;
    }
  }

  const actionClass =
    "min-h-11 rounded-lg border px-4 text-sm font-semibold transition-colors cursor-pointer";
</script>

<section class="mx-auto w-full max-w-3xl space-y-4">
  <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
    <h2 class="text-xl font-bold text-slate-50">{dateHeading}</h2>
    <div class="flex items-center gap-2">
      <button
        type="button"
        onclick={() => changeDay(-1)}
        aria-label={m.frontdesk_prev_day()}
        class="{actionClass} border-slate-700 bg-slate-800 text-slate-200 hover:bg-slate-700"
      >
        &larr;
      </button>
      <input
        type="date"
        bind:value={date}
        aria-label={m.frontdesk_pick_date()}
        class="min-h-11 min-w-0 flex-1 rounded-lg border border-slate-700 bg-slate-900 px-3 text-sm text-slate-100 sm:flex-none"
      />
      <button
        type="button"
        onclick={() => changeDay(1)}
        aria-label={m.frontdesk_next_day()}
        class="{actionClass} border-slate-700 bg-slate-800 text-slate-200 hover:bg-slate-700"
      >
        &rarr;
      </button>
      <button
        type="button"
        onclick={() => (date = getTodayDateString())}
        class="{actionClass} border-sky-500/40 bg-sky-500/10 text-sky-300 hover:bg-sky-500/20"
      >
        {m.frontdesk_today()}
      </button>
    </div>
  </div>

  <dl class="grid grid-cols-2 gap-2 sm:grid-cols-4">
    {#each [[m.frontdesk_stat_total(), stats.total], [m.frontdesk_stat_waiting(), stats.waiting], [m.frontdesk_stat_upcoming(), stats.upcoming], [m.frontdesk_stat_done(), stats.done]] as [label, value]}
      <div class="rounded-xl border border-slate-800 bg-slate-900/70 px-3 py-2">
        <dt class="text-xs font-medium text-slate-400">{label}</dt>
        <dd class="text-2xl font-bold text-slate-50">{value}</dd>
      </div>
    {/each}
  </dl>

  {#if dayAppointments.length === 0}
    <div class="rounded-xl border border-slate-800 bg-slate-900/70">
      <EmptyState title={m.frontdesk_empty_title()} subtitle={m.frontdesk_empty_sub()} />
    </div>
  {:else}
    <ul class="space-y-3">
      {#each dayAppointments as appt (appt.id)}
        {@const patient = findPatient(appt.patient_id)}
        <li class="rounded-xl border border-slate-800 bg-slate-900/70 p-4 shadow-sm">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <p class="text-sm font-semibold text-slate-300">
              {formatTime(appt.start_time)} &ndash; {formatTime(appt.end_time)}
            </p>
            <StatusBadge variant={appt.status} label={statusLabel(appt.status)} size="lg" />
          </div>
          <p class="mt-1 text-lg font-semibold text-slate-50">{patientName(appt.patient_id)}</p>
          <p class="text-sm text-slate-400">
            {[appt.reason, providerName(appt.provider_id), operatoryName(appt.operatory_id)]
              .filter(Boolean)
              .join(" · ")}
          </p>

          <div class="mt-3 flex flex-wrap gap-2">
            {#if appt.status === "scheduled"}
              <button
                type="button"
                onclick={() => onupdatestatus(appt.id, "confirmed")}
                class="{actionClass} border-blue-500/40 bg-blue-500/10 text-blue-300 hover:bg-blue-500/20"
              >
                {m.appts_action_confirm()}
              </button>
            {/if}
            {#if appt.status === "scheduled" || appt.status === "confirmed"}
              <button
                type="button"
                onclick={() => onupdatestatus(appt.id, "arrived")}
                class="{actionClass} border-amber-500/40 bg-amber-500/10 text-amber-300 hover:bg-amber-500/20"
              >
                {m.appts_action_arrived()}
              </button>
            {/if}
            {#if appt.status === "scheduled" || appt.status === "confirmed" || appt.status === "arrived"}
              <button
                type="button"
                onclick={() => onupdatestatus(appt.id, "cancelled")}
                class="{actionClass} border-rose-500/40 bg-rose-500/10 text-rose-300 hover:bg-rose-500/20"
              >
                {m.appts_action_cancel()}
              </button>
            {/if}
            <button
              type="button"
              onclick={() => toggleDetails(appt)}
              aria-expanded={expandedId === appt.id}
              class="{actionClass} ml-auto border-slate-700 bg-slate-800 text-slate-200 hover:bg-slate-700"
            >
              {expandedId === appt.id ? m.frontdesk_details_hide() : m.frontdesk_details_show()}
            </button>
          </div>

          {#if expandedId === appt.id}
            <div class="mt-4 grid gap-4 border-t border-slate-800 pt-4 sm:grid-cols-2">
              <div class="space-y-2">
                {#if accountLoading}
                  <p class="text-sm text-slate-400">{m.common_loading()}</p>
                {:else if accountError}
                  <p class="text-sm text-rose-400" role="alert">{accountError}</p>
                {:else if account}
                  <p class="text-xs font-medium uppercase tracking-wide text-slate-400">
                    {account.outstanding < 0
                      ? m.frontdesk_balance_credit()
                      : balanceDue === 0
                        ? m.frontdesk_balance_zero()
                        : m.frontdesk_balance_due()}
                  </p>
                  <p
                    class="text-3xl font-bold {balanceDue > 0
                      ? 'text-amber-300'
                      : 'text-emerald-300'}"
                  >
                    {money(Math.abs(account.outstanding))}
                  </p>
                  <dl class="space-y-1 text-sm">
                    {#each accountRows as row}
                      <div class="flex justify-between gap-3">
                        <dt class="text-slate-400">{row.label}</dt>
                        <dd class="font-medium text-slate-100">{money(row.cents)}</dd>
                      </div>
                    {/each}
                  </dl>
                  <p class="text-sm text-slate-300">
                    {m.frontdesk_claims_pending({ count: account.pending_claims })}
                  </p>
                  {#if account.rejected_claims > 0}
                    <p class="text-sm text-rose-300">
                      {m.frontdesk_claims_rejected({ count: account.rejected_claims })}
                    </p>
                  {/if}
                  <p class="text-sm text-slate-400">
                    {account.last_payment_date
                      ? m.frontdesk_last_payment({ date: account.last_payment_date })
                      : m.frontdesk_no_payments()}
                  </p>
                  <p class="text-xs text-slate-500">{m.frontdesk_balance_note()}</p>
                {/if}
              </div>

              <div class="space-y-4 text-sm">
                <div>
                  <p class="text-xs font-medium uppercase tracking-wide text-slate-400">
                    {m.frontdesk_insurance()}
                  </p>
                  {#if patient?.insurance_carrier}
                    <p class="font-medium text-slate-100">{patient.insurance_carrier}</p>
                    <p class="text-slate-400">
                      {[
                        patient.insurance_policy_number &&
                          m.frontdesk_policy({ policy: patient.insurance_policy_number }),
                        patient.insurance_group_number &&
                          m.frontdesk_group({ group: patient.insurance_group_number }),
                      ]
                        .filter(Boolean)
                        .join(" · ")}
                    </p>
                  {:else}
                    <p class="text-slate-300">{m.frontdesk_no_insurance()}</p>
                  {/if}
                </div>

                <div>
                  <p class="text-xs font-medium uppercase tracking-wide text-slate-400">
                    {m.frontdesk_alerts()}
                  </p>
                  {#if patient?.medical_alerts?.length}
                    <div class="mt-1 flex flex-wrap gap-1.5">
                      {#each patient.medical_alerts as alert}
                        <span
                          class="rounded-full border border-rose-500/40 bg-rose-500/10 px-2.5 py-0.5 text-xs font-semibold text-rose-300"
                          >{alert}</span
                        >
                      {/each}
                    </div>
                  {:else}
                    <p class="text-slate-300">{m.frontdesk_none_on_file()}</p>
                  {/if}
                </div>

                {#if patient?.allergies?.length}
                  <div>
                    <p class="text-xs font-medium uppercase tracking-wide text-slate-400">
                      {m.frontdesk_allergies()}
                    </p>
                    <div class="mt-1 flex flex-wrap gap-1.5">
                      {#each patient.allergies as allergy}
                        <span
                          class="rounded-full border border-amber-500/40 bg-amber-500/10 px-2.5 py-0.5 text-xs font-semibold text-amber-300"
                          >{allergy}</span
                        >
                      {/each}
                    </div>
                  </div>
                {/if}

                <div>
                  <p class="text-xs font-medium uppercase tracking-wide text-slate-400">
                    {m.frontdesk_contact()}
                  </p>
                  {#if patient?.phone_primary}
                    <a
                      class="font-medium text-sky-300 underline"
                      href="tel:{patient.phone_primary}"
                    >
                      {m.frontdesk_phone()}: {patient.phone_primary}
                    </a>
                  {:else}
                    <p class="text-slate-300">{m.frontdesk_no_phone()}</p>
                  {/if}
                  {#if patient?.guarantor_name}
                    <p class="text-slate-300">
                      {m.frontdesk_guarantor()}: {patient.guarantor_name}{patient.guarantor_phone
                        ? `, ${patient.guarantor_phone}`
                        : ""}
                    </p>
                  {/if}
                </div>
              </div>

              <form
                class="space-y-3 border-t border-slate-800 pt-4 sm:col-span-2"
                onsubmit={(e) => recordPayment(e, appt.patient_id)}
              >
                <p class="text-xs font-medium uppercase tracking-wide text-slate-400">
                  {m.frontdesk_take_payment()}
                </p>
                <div class="grid gap-3 sm:grid-cols-3">
                  <label class="block text-sm text-slate-300">
                    {m.frontdesk_amount()}
                    <input
                      type="number"
                      inputmode="decimal"
                      min="0"
                      step="0.01"
                      bind:value={amount}
                      class="mt-1 block min-h-11 w-full rounded-lg border border-slate-700 bg-slate-950 px-3 text-base text-slate-100"
                    />
                  </label>
                  <label class="block text-sm text-slate-300">
                    {m.frontdesk_method()}
                    <select
                      bind:value={method}
                      class="mt-1 block min-h-11 w-full rounded-lg border border-slate-700 bg-slate-950 px-3 text-base text-slate-100"
                    >
                      {#each deskMethods as option}
                        <option value={option}>{paymentMethodLabel(option)}</option>
                      {/each}
                    </select>
                  </label>
                  <div class="flex items-end gap-2">
                    <button
                      type="button"
                      disabled={balanceDue <= 0}
                      onclick={() => (amount = (balanceDue / 100).toFixed(2))}
                      class="{actionClass} flex-1 border-slate-700 bg-slate-800 text-slate-200 hover:bg-slate-700 disabled:cursor-not-allowed disabled:opacity-40"
                    >
                      {m.frontdesk_pay_full()}
                    </button>
                  </div>
                </div>
                <button
                  type="submit"
                  disabled={saving}
                  class="{actionClass} w-full border-emerald-500/40 bg-emerald-500/15 text-emerald-300 hover:bg-emerald-500/25 disabled:opacity-50 sm:w-auto"
                >
                  {m.frontdesk_record_payment()}
                </button>
                {#if payMessage}
                  <p class="text-sm text-emerald-300" role="status">{payMessage}</p>
                {/if}
                {#if payError}
                  <p class="text-sm text-rose-400" role="alert">{payError}</p>
                {/if}
              </form>
            </div>
          {/if}
        </li>
      {/each}
    </ul>
  {/if}
</section>
