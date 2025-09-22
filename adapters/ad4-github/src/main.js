const core = require('@actions/core');
const github = require('@actions/github');
const artifact = require('@actions/artifact');
const crypto = require('crypto');
const fs = require('fs');
const path = require('path');
const axios = require('axios');

async function run() {
  try {
    const startTime = Date.now();
    
    // Get inputs
    const artifactPath = core.getInput('artifact-path');
    const ocxServerUrl = core.getInput('ocx-server-url');
    const apiKey = core.getInput('api-key');
    const stepName = core.getInput('step-name');
    const failOnError = core.getInput('fail-on-error') === 'true';

    core.info(`Starting OCX verification for: ${stepName}`);

    // Generate artifact hash
    const artifactHash = await generateArtifactHash(artifactPath);
    core.info(`Artifact hash: ${artifactHash}`);

    // Create verification context
    const context = {
      workflow: github.context.workflow,
      job: github.context.job,
      step: stepName,
      sha: github.context.sha,
      ref: github.context.ref,
      repository: github.context.repo,
      actor: github.context.actor,
      run_id: github.context.runId,
      run_number: github.context.runNumber,
      timestamp: new Date().toISOString()
    };

    // Generate request digest
    const requestBody = JSON.stringify({
      artifact_hash: artifactHash,
      context: context
    });
    const requestDigest = crypto.createHash('sha256').update(requestBody).digest('hex');

    // Send verification request to OCX server
    const verificationResponse = await axios.post(
      `${ocxServerUrl}/api/v1/execute`, 
      {
        artifact: artifactHash,
        input: Buffer.from(requestBody).toString('base64'),
        cycles: 10000,
        request_digest: requestDigest
      },
      {
        headers: {
          'Authorization': `Bearer ${apiKey}`,
          'Content-Type': 'application/json',
          'User-Agent': 'OCX-GitHub-Action/1.0'
        },
        timeout: 30000
      }
    );

    if (verificationResponse.status !== 200) {
      throw new Error(`OCX server returned ${verificationResponse.status}: ${verificationResponse.statusText}`);
    }

    const receipt = verificationResponse.data;
    core.info(`Verification successful. Receipt generated.`);

    // Create receipt file
    const receiptData = {
      receipt: receipt,
      context: context,
      metadata: {
        generated_at: new Date().toISOString(),
        github_run_id: github.context.runId,
        github_run_number: github.context.runNumber,
        verification_duration_ms: Date.now() - startTime
      }
    };

    const receiptPath = path.join(process.env.RUNNER_TEMP, 'ocx-receipt.json');
    fs.writeFileSync(receiptPath, JSON.stringify(receiptData, null, 2));

    // Calculate receipt hash
    const receiptHash = crypto.createHash('sha256')
      .update(fs.readFileSync(receiptPath))
      .digest('hex');

    // Upload receipt as GitHub artifact
    const artifactClient = artifact.create();
    const artifactName = `ocx-receipt-${github.context.job}-${github.context.runId}`;
    
    const uploadResult = await artifactClient.uploadArtifact(
      artifactName,
      [receiptPath],
      process.env.RUNNER_TEMP,
      {
        continueOnError: false
      }
    );

    // Set outputs
    core.setOutput('receipt-url', uploadResult.artifactUrl || 'N/A');
    core.setOutput('verification-status', 'success');
    core.setOutput('receipt-hash', receiptHash);

    // Add job summary
    await core.summary
      .addHeading('OCX Verification Complete')
      .addTable([
        [{data: 'Property', header: true}, {data: 'Value', header: true}],
        ['Step Name', stepName],
        ['Artifact Hash', artifactHash],
        ['Receipt Hash', receiptHash],
        ['Verification Time', `${Date.now() - startTime}ms`],
        ['Status', '✅ Success']
      ])
      .write();

    core.info(`OCX verification completed successfully in ${Date.now() - startTime}ms`);

  } catch (error) {
    core.error(`OCX verification failed: ${error.message}`);
    
    await core.summary
      .addHeading('OCX Verification Failed')
      .addCodeBlock(error.message, 'text')
      .write();

    core.setOutput('verification-status', 'failure');
    
    if (core.getInput('fail-on-error') === 'true') {
      core.setFailed(error.message);
    }
  }
}

async function generateArtifactHash(artifactPath) {
  const hash = crypto.createHash('sha256');
  
  if (fs.statSync(artifactPath).isDirectory()) {
    // Hash directory contents
    const files = getAllFiles(artifactPath);
    for (const file of files.sort()) {
      const relativePath = path.relative(artifactPath, file);
      const content = fs.readFileSync(file);
      hash.update(relativePath);
      hash.update(content);
    }
  } else {
    // Hash single file
    const content = fs.readFileSync(artifactPath);
    hash.update(content);
  }
  
  return hash.digest('hex');
}

function getAllFiles(dir) {
  const files = [];
  const items = fs.readdirSync(dir);
  
  for (const item of items) {
    const fullPath = path.join(dir, item);
    if (fs.statSync(fullPath).isDirectory()) {
      files.push(...getAllFiles(fullPath));
    } else {
      files.push(fullPath);
    }
  }
  
  return files;
}

if (require.main === module) {
  run();
}

module.exports = { run };
