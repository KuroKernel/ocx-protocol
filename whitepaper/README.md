# OCX Whitepaper — build instructions

The paper is one self-contained LaTeX file: `paper.tex`. Bibliography is inline (no separate `.bib`). Target output: ~10 pages, single-column, 10 pt, 1-inch margins. Render to PDF.

You have three sensible options. Pick one.

---

## Option A — Overleaf (recommended; no install, web-based)

Easiest by a wide margin. Works from any browser. Free tier is sufficient.

1. Go to [https://www.overleaf.com](https://www.overleaf.com), sign in (or sign up — Google login works).
2. Top-left → **New Project** → **Blank Project**. Name it `ocx-whitepaper`.
3. Delete the auto-generated `main.tex`.
4. Click **Upload** (top-left of the file panel) → upload `whitepaper/paper.tex` from this repo.
5. Top-right of the editor: **Menu** → **Main document** → set to `paper.tex`.
6. Click **Recompile** (green button, top of the PDF preview pane). First compile takes ~30 seconds; subsequent ones are instant.

To export the final PDF: top-right → **Download PDF**.
To edit collaboratively: top-right → **Share** → invite by email. Co-authors edit in the same browser, no merge conflicts.

This is what I'd use. No software install. No font hunting. No package mismatches.

---

## Option B — Local TeX Live (if you want to compile offline)

On your Pop!_OS workstation:

```bash
sudo apt install texlive-latex-recommended texlive-latex-extra texlive-fonts-recommended
cd /home/kurokernel/Desktop/AXIS/ocx-protocol/whitepaper
pdflatex paper.tex && pdflatex paper.tex
```

Two `pdflatex` runs are needed — first builds the table of internal references, second resolves them. Output: `paper.pdf` in the same directory.

Open the PDF with whatever PDF reader you have (`xdg-open paper.pdf` or just double-click in Files).

For a live-rebuild loop while editing:

```bash
sudo apt install latexmk
cd whitepaper
latexmk -pdf -pvc paper.tex
```

`latexmk -pvc` watches the file and rebuilds + refreshes the PDF viewer on every save.

---

## Option C — Microsoft Word (only if a journal or co-author requires it)

LaTeX → Word is always lossy. Math, tables, and bibliography lose fidelity. Only do this if forced.

```bash
sudo apt install pandoc
cd whitepaper
pandoc paper.tex -o paper.docx --bibliography=none
```

Open `paper.docx` in LibreOffice Writer (`libreoffice paper.docx`) or Microsoft Word. Expect to fix:
- Lemma/Corollary environments (Pandoc strips the theorem styling)
- The two main tables (column alignment usually breaks)
- Inline `\texttt{}` code spans (some get lost)

If you genuinely need Word as the final format, consider drafting in Word from scratch using `paper.tex` as a content reference. Faster than fixing Pandoc output.

---

## Submission format

For arXiv: upload `paper.tex` directly. arXiv compiles it server-side. Their LaTeX setup matches Overleaf's, so what you see in Overleaf is what arXiv produces.

For USENIX Security / IEEE S&P / CCS: each conference has its own LaTeX class file (usenix-2024, IEEEtran, sig-alternate). When you pick a venue, swap `\documentclass[10pt]{article}` for the venue's class and re-compile. Most of the body stays the same; you'll re-do the title block.

For a non-academic public release (blog, company website, LinkedIn): `paper.pdf` from Overleaf. Done.

---

## What's in the paper

10 pages, 9 sections + references. Section ordering matches the structure you specified:

1. Introduction (~1.5 pp)
2. Related Work (~1 pp)
3. Protocol (~1 pp)
4. Implementation (~1 pp)
5. Empirical Results (~2 pp) — cross-language parity, verification perf, byte-identical inference matrix, long-run stability, cross-session reproducibility
6. Soundness (~2 pp) — threat model, hypergeometric lemma, replay-irrelevance lemma, Monte Carlo validation, comparison table
7. Limitations (~0.5 pp)
8. Conclusion (~0.25 pp)
9. References (~0.5 pp)

Every empirical claim is sourced to a specific committed file in this repo. Every lemma has a proof.

## What still needs your hand

1. **Title and author block.** Currently: "Aishwary H., hhaishwary@gmail.com, OCX Protocol". If you want a different author name, affiliation, or co-author, edit `\author{...}` near the top of `paper.tex`.
2. **Date.** Currently `April 2026`. Change `\date{...}` if you want a specific submission date.
3. **Abstract.** Drafted. Read it once and decide if it sells the work the way you want. Word count is ~210, on target.
4. **Open questions in the conclusion.** Both are honest open problems worth working on. If you want a third one, add it.
5. **#4 (AMD MI300X) backfill.** Once funds land and the experiment runs, add one row to Table 1 (the determinism matrix) and one paragraph to §5 noting that cross-vendor receipts round-trip through the canonical verifier byte-for-byte while the inference outputs themselves differ as expected. Single-paragraph addition.
6. **Acknowledgments.** Currently mentions JarvisLabs and the model providers. Add anyone else who deserves credit.

## After Overleaf compiles cleanly

1. Read end-to-end once. Check claims against current numbers in TEST_RESULTS.md.
2. Run a spelling pass (Overleaf has built-in spell-check; toggle in Menu → Spell check).
3. Get a pre-read from one trusted technical reader. Their first reaction tells you whether the abstract and intro are working.
4. arXiv submit: register at arxiv.org, category `cs.CR` (cryptography/security), cross-list `cs.LG`. Upload the .tex; they handle compilation. Within 24 hours you have a citable URL.

If anything in the LaTeX fails to compile in Overleaf, paste the error here and I'll fix it.
