// Tailwind class-name combinator. Re-export clsx with the standard `cn` name
// many React projects use; we don't pull in tailwind-merge yet — if we hit
// duplicate-class-collision issues we can swap in.
export { clsx as cn } from 'clsx'
