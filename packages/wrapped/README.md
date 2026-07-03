# seshy wrapped

**Your AI-coding life, on a card you'll actually want to post.**

```bash
npx seshy-wrapped
```

No install. No account. No API key. **100% local — your sessions never leave
your machine.**

`seshy-wrapped` reads the session logs your coding agents already keep on
disk — **Claude Code, Codex, Gemini CLI, OpenCode** (plus pi and Droid) — and
renders a synthwave stat card: how many prompts you fired, how many lines of
code it wrote for you, how many times you swore at it, how many times it
caved and said *"you're right"*, your 3am prompt count, and a
tongue-in-cheek **delegation grade**.

![example card](docs/example.png)

*(Example rendered from the synthetic test fixtures — run it yourself for
your real numbers.)*

## The hero move

```bash
npx seshy-wrapped --model fable
```

Filters to work answered by Fable 5 and retitles the card `FABLE 5 · 2026`.
Works for any model: `--model opus-4-8`, `--model gpt-5.4`.

## Cuts and themes

One dataset, four cuts, six looks:

```bash
npx seshy-wrapped --cut machine --theme receipt   # the itemized receipt
npx seshy-wrapped --cut chaos                     # "wait", "actually", caps-lock rage
npx seshy-wrapped --cut agent                     # what the model kept saying to you
npx seshy-wrapped --all-cuts --theme terminal     # every cut, terminal look
```

| Cut | Hero stat |
| --- | --- |
| `classic` | prompts fired |
| `machine` | lines of code it wrote for you |
| `chaos` | times you said "actually" |
| `agent` | times it said "let me…" |

Themes: `sunset` (default), `terminal`, `starfield`, `receipt`, `billboard`, `crt`.

## All options

```
-a, --agent <name>    claude | codex | gemini | opencode | pi | droid | all
-m, --model <substr>  only work answered by matching models
    --since <date>    only sessions after YYYY-MM-DD
-p, --period <win>    week | month | year | all
-c, --cut <cut>       classic | machine | chaos | agent
-t, --theme <look>    sunset | terminal | starfield | receipt | billboard | crt
    --all-cuts        render every cut in the chosen theme
-h, --handle <@you>   handle printed on the card
    --json            emit stats.json only
-o, --out <dir>       output directory (default ./seshy-wrapped)
    --open            open the card when done (macOS)
```

Installed seshy? `seshy wrapped` does the same thing.

## Deterministic, auditable, local

- Same logs in, byte-identical `stats.json` and pixel-identical card out —
  enforced in CI by running the whole pipeline twice and diffing.
- Every stat's exact definition (regex, roles scanned, exclusions) is in
  [STATS.md](STATS.md). Grep your own logs; the numbers will hold up.
- Zero network calls. Zero telemetry. Read the source — it's small.

## Privacy

It reads files, counts words, writes a PNG. Prompts, code, and paths never
leave your machine and never appear on the card (only aggregate numbers and
your top project's basename).

## Want this live, all the time?

[seshy](https://github.com/buddyh/seshy) keeps every one of these sessions
searchable and resumable from your terminal, every day. This card is what
seshy sees.

## License

MIT
