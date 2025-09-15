# OCX Protocol Waitlist Form Setup

## Phase 1: Form Submission Options

### Option 1: Formspree (Recommended - Free)
1. Go to https://formspree.io
2. Sign up for a free account
3. Create a new form
4. Copy the form ID (looks like: `f/abc123def456`)
5. Replace `YOUR_FORM_ID` in `request.html` with your actual form ID
6. Formspree will handle email delivery automatically

### Option 2: Google Forms
1. Go to https://forms.google.com
2. Create a new form with fields: Email, Name, Organization, Use Case
3. Get the form URL
4. Replace the form action in `request.html` with the Google Forms URL
5. Set method to "GET" instead of "POST"

### Option 3: Simple Mailto (Fallback)
1. Replace the form action with: `mailto:hello@ocx.world`
2. Add subject line: `?subject=OCX Protocol Access Request`
3. This will open the user's email client

## Current Status
- ✅ Landing page created (`index.html`)
- ✅ Waitlist form created (`request.html`)
- ✅ Links properly connected
- ⏳ Form submission needs to be configured

## Testing
1. Visit http://localhost:8080 to see the landing page
2. Click "Request Access" to test the form
3. Verify the form submission works with your chosen method

## Deployment
Upload both files to your ocx.world hosting:
- `index.html` → root directory
- `request.html` → root directory
