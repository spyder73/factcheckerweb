# @alethea/mobile

Expo Router app for iOS + Android. This is **scaffolding** (Phase 6) — the
screens render, the API client compiles, share-intent is wired, but a few
pieces need real work before TestFlight.

## What's here

| File | Purpose |
| --- | --- |
| `app.json` | Expo config — bundle IDs, share-intent activation rules, iOS NSCameraUsage strings |
| `app/_layout.tsx` | Root `Stack` navigator + dark theme |
| `app/index.tsx` | Home — paste/check screen, picks up shared URLs via `expo-share-intent` |
| `app/check/[id].tsx` | Verdict screen — polls `/api/check/{id}` until done |
| `app/history.tsx` | History list (calls `/api/me/checks` — endpoint TODO) |
| `app/login.tsx` | Sign-in (assumes the API supports token-in-header for mobile clients) |
| `app/settings.tsx` | Account, GDPR data-export/delete, sign-out |
| `lib/api.ts` | Fetch wrapper with `X-Session-Token` + `X-CSRF-Token` headers |
| `lib/theme.ts` | Dark-only color tokens + verdict labels matching the web design system |
| `eas.json` | EAS Build profiles: `development`, `preview`, `production` |

## What needs human work

1. **Apple Developer + Google Play accounts** — see top-level human TODO.
2. **EAS project init** — run `eas init` to register the project on EAS and
   get an `EAS_PROJECT_ID`, then add `"extra": {"eas": {"projectId": "..."}}`
   to `app.json`.
3. **App icons + splash** — drop real PNGs at `assets/{icon,splash,favicon}.png`
   (1024×1024 for icon).
4. **Backend: token-in-header auth path**. The mobile client expects
   `/auth/login` to return `{user, sessionToken, csrfToken}` and to accept
   `X-Session-Token` on subsequent requests. The web SPA uses HttpOnly
   cookies; this is a parallel path. Add `auth.OptionalToken(pool)` that
   reads `X-Session-Token` and falls back to the cookie middleware. Mark
   sessions with a `client` column so we can revoke per-device.
5. **Backend: `/api/me/checks` endpoint** for the history list.
6. **Backend: data-export download**. The current `/api/me/data-export`
   returns a JSON file; mobile needs a native share/download UX
   (e.g. write to `expo-file-system` + `expo-sharing`).
7. **Push notifications** — `expo-notifications` setup + a webhook from
   the backend when a long-running check completes.
8. **iOS share extension** — `expo-share-intent` handles the basic
   activation but you'll need to test on a real device; the simulator
   doesn't reliably show "Share to Alethea" in the share sheet.
9. **Android intent handling** — verify the intent filter in `app.json`
   actually surfaces the app in the share sheet for `text/plain` and
   `image/*` shares.
10. **Visual polish** — once the flow works end-to-end, hire a designer
    or use the web `tokens.ts` as a starting palette for refinement.

## Running

```sh
cd apps/mobile
npm install
npm start            # opens Expo Dev Tools
npm run ios          # builds + runs in iOS simulator
npm run android      # ditto Android
```

> **Heads up:** this directory has no `node_modules` committed and no
> lockfile. Install fresh — `npm install` here, NOT `npm install` at the
> repo root (which would mix with apps/web's dependencies).

## Sharing types with apps/web

The `@alethea/shared-types` package under `packages/shared-types/` is the
canonical source for `Verdict`, `CheckRow`, `JudgeOutput`, etc. The web app
still has its own copy at `apps/web/src/types/api.ts` for now — switch it
over to importing from `@alethea/shared-types` in a follow-up commit.
