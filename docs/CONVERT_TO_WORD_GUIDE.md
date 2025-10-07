# How to Convert Markdown to Word

All documentation has been created in **Markdown format (.md)** which is easily convertible to Microsoft Word (.docx).

---

## Option 1: Using Pandoc (Best Quality)

### Install Pandoc on Pop OS:
```bash
sudo apt update
sudo apt install pandoc
```

### Convert Single File:
```bash
# White Paper
pandoc OCX_PROTOCOL_WHITEPAPER.md -o OCX_WhitePaper.docx

# Technical Architecture
pandoc TECHNICAL_ARCHITECTURE.md -o OCX_TechnicalArchitecture.docx

# Audit Reports
pandoc COMPREHENSIVE_AUDIT_REPORT.md -o OCX_AuditReport.docx
pandoc AUDIT_SUMMARY.md -o OCX_AuditSummary.docx
```

### Convert All at Once:
```bash
#!/bin/bash
# Convert all markdown documents to Word

pandoc OCX_PROTOCOL_WHITEPAPER.md -o OCX_WhitePaper.docx
pandoc TECHNICAL_ARCHITECTURE.md -o OCX_TechnicalArchitecture.docx
pandoc COMPREHENSIVE_AUDIT_REPORT.md -o OCX_AuditReport.docx
pandoc AUDIT_SUMMARY.md -o OCX_AuditSummary.docx
pandoc DEPLOYMENT_GUIDE.md -o OCX_DeploymentGuide.docx
pandoc README.md -o OCX_README.docx

echo "✅ All documents converted to Word format"
```

### Advanced Conversion (With Styling):
```bash
# Create a reference.docx with your preferred styles first
pandoc --reference-doc=reference.docx OCX_PROTOCOL_WHITEPAPER.md -o OCX_WhitePaper.docx
```

---

## Option 2: Using LibreOffice (Pre-installed on Pop OS)

### Method A: GUI
1. Open LibreOffice Writer
2. File → Open
3. Select the `.md` file
4. File → Export As → Export as PDF or Save As → .docx

### Method B: Command Line
```bash
# Convert to DOCX
libreoffice --headless --convert-to docx OCX_PROTOCOL_WHITEPAPER.md

# Convert to PDF
libreoffice --headless --convert-to pdf OCX_PROTOCOL_WHITEPAPER.md
```

---

## Option 3: Google Docs (Cloud)

1. Upload `.md` file to Google Drive
2. Right-click → Open with → Google Docs
3. File → Download → Microsoft Word (.docx)

**Pros**: Works anywhere, no installation
**Cons**: Requires internet, some formatting may be lost

---

## Option 4: Transfer to Windows Laptop

### Transfer Files:
```bash
# Option A: USB Drive
cp *.md /media/your-usb-drive/

# Option B: SCP (if Windows has SSH)
scp *.md user@windows-laptop:/path/

# Option C: Cloud (Dropbox, Google Drive, etc.)
# Just upload and download on Windows
```

### Convert on Windows:
1. **Pandoc for Windows**: https://pandoc.org/installing.html
2. **Typora**: https://typora.io/ (Markdown editor with Word export)
3. **MarkdownPad**: http://markdownpad.com/
4. **Microsoft Word 2016+**: Can directly open `.md` files

---

## Recommended Workflow

### For Professional Documents:

```bash
# 1. Install pandoc
sudo apt install pandoc texlive-latex-base

# 2. Convert with professional formatting
pandoc OCX_PROTOCOL_WHITEPAPER.md \
  --from markdown \
  --to docx \
  --output OCX_WhitePaper_v1.0.docx \
  --toc \
  --toc-depth=3 \
  --highlight-style=tango

# 3. Open in LibreOffice to refine
libreoffice OCX_WhitePaper_v1.0.docx

# 4. Save final version
```

---

## Documents Created

All ready for conversion in:
`/home/kurokernel/Desktop/AXIS/ocx-protocol/`

| File | Description | Pages | Size |
|------|-------------|-------|------|
| `OCX_PROTOCOL_WHITEPAPER.md` | Complete white paper (technical + business) | ~30 | ~60KB |
| `TECHNICAL_ARCHITECTURE.md` | Deep technical documentation | ~25 | ~55KB |
| `COMPREHENSIVE_AUDIT_REPORT.md` | Full codebase audit | ~20 | ~45KB |
| `AUDIT_SUMMARY.md` | Executive summary audit | ~8 | ~20KB |
| `DEPLOYMENT_GUIDE.md` | Production deployment guide | ~6 | ~15KB |
| `README.md` | Quick start guide | ~8 | ~12KB |

---

## Quick Commands Cheat Sheet

```bash
# Install pandoc
sudo apt install pandoc

# Convert white paper
pandoc OCX_PROTOCOL_WHITEPAPER.md -o WhitePaper.docx

# Convert with table of contents
pandoc OCX_PROTOCOL_WHITEPAPER.md -o WhitePaper.docx --toc

# Convert to PDF instead
pandoc OCX_PROTOCOL_WHITEPAPER.md -o WhitePaper.pdf

# Batch convert all
for file in *.md; do
    pandoc "$file" -o "${file%.md}.docx"
done
```

---

## Troubleshooting

### "Command not found: pandoc"
```bash
sudo apt update
sudo apt install pandoc
```

### "PDF conversion failed"
```bash
# Install LaTeX for PDF generation
sudo apt install texlive-latex-base texlive-fonts-recommended
```

### "Code blocks not formatted"
```bash
# Use syntax highlighting
pandoc input.md -o output.docx --highlight-style=tango
```

### "Tables broken"
```bash
# Use advanced table support
pandoc input.md -o output.docx --columns=100
```

---

## Next Steps

1. **On Pop OS**: Convert documents using pandoc
2. **Review**: Open in LibreOffice to check formatting
3. **Polish**: Add company logo, adjust fonts/colors
4. **Export**: Save as final .docx or PDF
5. **Transfer**: Move to Windows laptop if needed

---

## Contact Support

If conversion issues occur:
- **Pandoc Docs**: https://pandoc.org/MANUAL.html
- **LibreOffice Help**: Press F1 in LibreOffice Writer

---

**All documentation is now ready for deployment, presentation, or publication!**
