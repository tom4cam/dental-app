# Dental app: notes for Claude Code

This private repository started as a copy of LibreDental (https://github.com/Cojekt/LibreDental, Apache-2.0). Tom Caswell and Michael Atkinson are turning it into a dental practice management app that works for a US practice. Michael knows how a front desk and a clinic really run. Ask him what an office needs, and explain your choices in plain English, because he is learning Claude Code as he goes.

Read the upstream project's own rules first. They cover the layout, how to run it, and the care medical software needs:

@AGENTS.md

## Rules for this project

- No real patient data, ever. No names, birthdates, phone numbers, insurance IDs or screenshots from a real practice in code, tests, fixtures, prompts, commit messages or issues. Use `task demo` data or made-up patients.
- Never copy code from Apexo (https://github.com/elselawi/apexo). It is GPL-3.0, and copying it would force this whole project under the GPL and block a commercial product. Learning from its ideas is fine. Copying its code is not. The same goes for any GPL project, and for Open Dental, which has been proprietary since version 24.4.
- Keep the Apache-2.0 license and the existing copyright notices. The README calls the project "LibreDental" with a trademark mark. Pick our own product name before any commercial release, and check the trademark question first.
- CDT procedure codes belong to the American Dental Association. Only the small sample set ships here. Don't paste a full code list into the repository. A full set needs a license from the ADA.
- Work on a branch named michael/topic or tom/topic, make small commits, and open a pull request. Don't push straight to main.
- Stop and ask before you change the database schema or add a migration, touch the audit trail, add a dependency, or change how stored money or dates are read.
- Run `task test` and `task format` before opening a pull request.

## What is missing

This list comes from Tom's review on October 19, 2026. Check each item by reading the code before you rely on it.

- A patient ledger, statements and aging reports.
- Insurance eligibility checks and electronic payment posting (835). Claims already go out as 837D through Stedi.
- Encryption of the SQLite database at rest. Staff PINs look like plain four-digit values, and the LAN server speaks plain HTTP.
- The full CDT code set, and a US-ready fee schedule and insurance workflow.

Decide the order with Michael: what the front desk needs first matters more than what is technically interesting.

## Running it on a Chromebook

A Chromebook runs this inside its Linux environment. The README lists the packages (Go, Node 20 or newer, and the GTK and WebKit libraries for Linux). If the desktop window doesn't work in that environment, run the headless server with `task server` and open it in Chrome at port 4242. Ask Claude to walk you through the install one step at a time.
