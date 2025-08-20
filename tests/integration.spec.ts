import { test, expect } from '@playwright/test';
import { spawn, ChildProcess } from 'child_process';
import { execSync } from 'child_process';
import path from 'path';

let betanetProcess: ChildProcess | null = null;

test.describe('Betanet Platform Tests', () => {
  test.beforeAll(async () => {
    // Start betanet process
    const betanetPath = path.join(__dirname, '..', 'bin', 'betanet');
    betanetProcess = spawn(betanetPath, ['start', '-node-port', '4001'], {
      cwd: path.join(__dirname, '..'),
      stdio: 'pipe'
    });

    // Wait for startup
    await new Promise(resolve => setTimeout(resolve, 5000));
  });

  test.afterAll(async () => {
    if (betanetProcess) {
      betanetProcess.kill();
    }
  });

  test('Browser interface loads correctly', async ({ page }) => {
    await page.goto('http://localhost:8080');
    
    await expect(page).toHaveTitle(/Betanet/);
    await expect(page.locator('h1')).toContainText('Betanet');
    await expect(page.locator('text=Decentralized Web Browser')).toBeVisible();
    
    // Test the site input functionality
    await page.fill('#siteInput', 'test123');
    await expect(page.locator('#siteInput')).toHaveValue('test123');
  });

  test('Wallet interface loads correctly', async ({ page }) => {
    await page.goto('http://localhost:8081');
    
    await expect(page).toHaveTitle(/Betanet Wallet/);
    await expect(page.locator('h1')).toContainText('Betanet Wallet');
    await expect(page.locator('text=Manage wallets, sites')).toBeVisible();
    
    // Test wallet creation functionality
    await page.click('button:has-text("Create New Wallet")');
    
    // Wait for API response
    await page.waitForTimeout(2000);
    
    // Check if result is displayed
    const result = page.locator('#walletResult');
    await expect(result).toBeVisible();
  });

  test('Node management interface loads correctly', async ({ page }) => {
    await page.goto('http://localhost:8082');
    
    await expect(page).toHaveTitle(/Node Management/);
    await expect(page.locator('h1')).toContainText('Node Management');
    await expect(page.locator('text=Monitor and manage')).toBeVisible();
    
    // Check node status
    const nodeStatus = page.locator('#nodeStatus');
    await expect(nodeStatus).toContainText('Online');
    
    // Check node ID is displayed
    const nodeId = page.locator('#nodeId');
    await expect(nodeId).not.toHaveText('Loading...');
  });

  test('Wallet API endpoints work', async ({ request }) => {
    // Test status endpoint
    const statusResponse = await request.get('http://localhost:8081/api/status');
    expect(statusResponse.ok()).toBeTruthy();
    
    const statusData = await statusResponse.json();
    expect(statusData.server).toBe('betanet-wallet');
    expect(statusData.node_id).toBeDefined();
    
    // Test wallet creation
    const createResponse = await request.post('http://localhost:8081/api/wallet/new');
    expect(createResponse.ok()).toBeTruthy();
    
    const createData = await createResponse.json();
    expect(createData.success).toBe(true);
    expect(createData.mnemonic).toBeDefined();
    expect(createData.wallet).toBeDefined();
  });

  test('Node API endpoints work', async ({ request }) => {
    // Test node status
    const statusResponse = await request.get('http://localhost:8082/api/node/status');
    expect(statusResponse.ok()).toBeTruthy();
    
    const statusData = await statusResponse.json();
    expect(statusData.server).toBe('betanet-node-ui');
    expect(statusData.node_id).toBeDefined();
    expect(statusData.status).toBe('online');
    
    // Test peers endpoint
    const peersResponse = await request.get('http://localhost:8082/api/node/peers');
    expect(peersResponse.ok()).toBeTruthy();
    
    const peersData = await peersResponse.json();
    expect(peersData.success).toBe(true);
    expect(peersData.count).toBeDefined();
  });

  test('Storage and domain management', async ({ request }) => {
    // Test domains list
    const domainsResponse = await request.get('http://localhost:8081/api/domains/list');
    expect(domainsResponse.ok()).toBeTruthy();
    
    const domainsData = await domainsResponse.json();
    expect(domainsData.success).toBe(true);
    expect(domainsData.domains).toBeDefined();
  });

  test('All interfaces are accessible simultaneously', async ({ browser }) => {
    // Open multiple pages simultaneously
    const context1 = await browser.newContext();
    const context2 = await browser.newContext();
    const context3 = await browser.newContext();
    
    const page1 = await context1.newPage();
    const page2 = await context2.newPage();
    const page3 = await context3.newPage();
    
    // Load all interfaces
    await Promise.all([
      page1.goto('http://localhost:8080'),
      page2.goto('http://localhost:8081'),
      page3.goto('http://localhost:8082')
    ]);
    
    // Verify all loaded correctly
    await expect(page1.locator('h1')).toContainText('Betanet');
    await expect(page2.locator('h1')).toContainText('Betanet Wallet');
    await expect(page3.locator('h1')).toContainText('Node Management');
    
    await context1.close();
    await context2.close();
    await context3.close();
  });
});