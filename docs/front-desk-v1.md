# Front Desk, first version

The **Front Desk** tab is a single page for the front-desk day: the schedule for one day, a check-in card for each patient, and a quick way to take a payment. It was built from the outline in [front-desk-day.md](front-desk-day.md), and Michael hasn't tried it yet.

It runs in a browser, so it works on Windows, Mac and Chromebook, and the layout is built for a phone screen too. It adds no database tables, no migration and no new dependency.

## What it does

- **The day's schedule.** Pick a date with the arrows or the date box, or press Today. Each appointment shows the time, the patient, the reason, the dentist and the chair. Four counters at the top show how many appointments there are, how many patients are waiting (arrived or in the chair), how many are still to come, and how many are done. Cancelled visits aren't counted.
- **One-tap status buttons.** Confirm, Arrived and Cancel use the same status change as the Appointments tab, so both tabs always agree.
- **A check-in card per patient** ("Check-in details"):
  - the balance, split into billed, paid by the patient, paid by insurance and written off, with the number of claims still waiting on insurance and the date of the last payment;
  - the insurance carrier, policy and group, or a note when none is on file;
  - medical alerts, and allergies when any are stored (no screen can enter allergies yet, so a blank doesn't mean the patient has none; adding that field to the patient form is a good next step);
  - the phone number (tap to call on a phone), and the guarantor's name and phone if there is one.
- **Take payment.** Cash, check or card, an amount, and a button that fills in the whole balance. It uses the same call as Billing & Claims > Payments, so the payment shows up there and in the audit trail.

## What the balance means

The balance is the same number Billing & Claims uses: everything billed on claims, minus every payment recorded for the patient. Two things follow from that, and the card says so on screen:

- A claim counts as billed from the moment it exists, even as a draft, and the amount insurance is expected to pay isn't subtracted until someone records an insurance payment. So a patient whose insurance hasn't paid yet can show a large balance that is mostly the insurer's.
- Payments aren't matched to individual charges, so there's no aging by invoice. A true ledger, statements and aging reports are still missing (issue #10).

A negative balance means the patient has paid more than has been billed, and the card calls it a credit.

## What it can't do yet

These all need a new database table or column, so they wait for a decision:

- payment plans, and a record of who was owed what when;
- a check-in time, and a reason for a cancellation;
- closing out the day, and a day sheet that separates cash, checks and cards;
- family and guarantor accounts that roll balances together;
- an "insurance verified on" date.

## Using it on a phone

`task server` listens on the whole network by default. A phone on the same Wi-Fi can open `http://<the computer's address>:4242`. That connection is plain HTTP, so the PIN and everything on screen can be read by anyone on that network. Until the server is set up with a secure connection, use this with demo data only, never real patients.

To keep the server on the one computer, start it with `LIBREDENTAL_HOST=127.0.0.1`.

## For developers

- The new read-only call is `BillingService.GetPatientAccount(token, patientID)`. It needs a staff session, logs a read in the audit trail, and returns `domain.PatientAccount`. Its outstanding amount matches `GetPatientBalance` by design, and a test checks that.
- The page is `frontend/src/views/FrontDeskView.svelte`. It reuses the appointments and patients the app already loads, so it adds no extra requests for the schedule.
- The demo data's "today" appointments are dated in 2027. To see them, open the page and choose June 15, 2027.
- The header and tab bar were made to fit a phone: the tab row scrolls sideways, and the staff button shrinks. The other tabs weren't redesigned, so tables on the older screens are still wide.
