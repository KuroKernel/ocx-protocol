#!/bin/bash
# Convert all OCX documentation to Word format
# Run this script on Pop OS to generate .docx files

echo "🔄 Converting OCX Protocol Documentation to Word format..."
echo ""

# Check if pandoc is installed
if ! command -v pandoc &> /dev/null; then
    echo "❌ Pandoc not found. Installing..."
    sudo apt update && sudo apt install -y pandoc
fi

echo "✅ Pandoc installed"
echo ""

# Convert each document
echo "📄 Converting White Paper..."
pandoc OCX_PROTOCOL_WHITEPAPER.md -o OCX_WhitePaper.docx --toc --toc-depth=3

echo "📄 Converting Technical Architecture..."
pandoc TECHNICAL_ARCHITECTURE.md -o OCX_TechnicalArchitecture.docx --toc --toc-depth=3

echo "📄 Converting Comprehensive Audit Report..."
pandoc COMPREHENSIVE_AUDIT_REPORT.md -o OCX_AuditReport_Full.docx --toc

echo "📄 Converting Audit Summary..."
pandoc AUDIT_SUMMARY.md -o OCX_AuditReport_Summary.docx

echo "📄 Converting Deployment Guide..."
pandoc DEPLOYMENT_GUIDE.md -o OCX_DeploymentGuide.docx

echo "📄 Converting README..."
pandoc README.md -o OCX_README.docx

echo ""
echo "✅ All documents converted successfully!"
echo ""
echo "Generated files:"
ls -lh *.docx | awk '{print "  ", $9, "(" $5 ")"}'
echo ""
echo "💡 Tip: Open in LibreOffice Writer to review and polish"
echo "📧 Transfer to your Windows laptop for final editing"
echo ""
echo "🎉 Done!"
