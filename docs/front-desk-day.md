# The front-desk day

Status: first draft, October 23, 2026. This is the write-up behind issue #9.

Where it comes from: Tom replied for himself and Michael to the request for a front-desk day. Michael hasn't described the steps in his own words yet. So the step list below is the outline we asked him to cover, which Tom confirmed is a good starting point. It isn't Michael's account of what he does. Everything marked "to confirm" needs his detail before we build to it.

## What we know

- **Software today: none.** Michael's practice is new, so there is no existing practice-management software to import from or to match. For a first version that means no data migration and no old workflow to copy.
- **Computers: a mix.** The front desk will use Windows PCs, Macs and Chromebooks. Ideally a phone browser can open the pages too, "in a pinch". So the first version runs in a browser (`task server`), nothing depends on a desktop-only feature, and every page has to work at phone width.
- **Direction:** build a first version from this outline, then check it with Michael.

## The steps

| # | Step | What we know | To confirm with Michael |
| --- | --- | --- | --- |
| 1 | Opening the office and looking at the day's schedule | On the outline | What he checks first (who is coming in, which chairs, which providers) |
| 2 | Check-in | On the outline | See below |
| 3 | Rescheduling and cancellations | On the outline | How a cancelled slot gets refilled |
| 4 | Treatment plans, and telling the patient what they'll owe | On the outline | Where the estimate comes from (fee schedule, insurance) |
| 5 | Taking payment at the desk, and payment plans | On the outline | Payment types (cash, card, check) and what a payment plan looks like |
| 6 | Insurance claims, from sending them to following up on rejections | On the outline | Which insurers, and how he chases a rejection today |
| 7 | Patient statements, and phone calls about balances | On the outline | How statements go out (mail, email, text) and when |
| 8 | Closing out the day | On the outline | What gets counted or reconciled at the end |

For each step we still need three things: what the person does, what they need to see on the screen, and what goes wrong today.

### Check-in: one example, made up

The request included one invented example, so treat it as an illustration of the level of detail, not as Michael's process:

- A patient arrives for a 9:00 appointment. The desk confirms contact details, asks whether insurance changed, scans a new card if it did, and tells the assistant the patient is here.
- The desk needs to see: the name, today's appointment (procedure, dentist, chair), the insurance and whether it has been verified, what the patient owed last time, and any medical alerts such as an allergy or a blood thinner.
- What goes wrong in the example: the balance is hard to find, and a changed insurance card means retyping everything by hand.

If Michael recognizes either problem, those are our first two candidates: show the balance at check-in, and make an insurance change easy to enter.

## Questions we still have

- What are the three things that waste the most time or cause the most mistakes at his front desk? (Still asked for in issue #9.)
- Does the practice take insurance, and if so, from how many companies? Or is some of it fee-for-service?
- How many dentists, chairs and front-desk staff are there? Does more than one person use the schedule at the same time?
- What payment types does the desk take, and which card terminal or processor, if any?
- How do patients get statements and reminders today (paper, email, text)?
- Which of the eight steps hurts most? That one decides what we build first.

## Privacy

Everything here is made up or general. No real patient names, birthdates, phone numbers or insurance numbers go in this file, in issues, or anywhere else in the project.
