import { test, expect } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';
import { execSync } from 'child_process';

// Config file path (relative to test file)
const CONFIG_FILE = path.join(__dirname, '../../config/proxy.e2e.with-whitelist.yaml');

// Helper to read config file
function readConfig(): string {
  return fs.readFileSync(CONFIG_FILE, 'utf-8');
}

// Helper to write config file
function writeConfig(content: string): void {
  fs.writeFileSync(CONFIG_FILE, content, 'utf-8');
}

// Helper to get container logs
function getContainerLogs(): string {
  try {
    return execSync('docker logs e2e-proxy-app-with-whitelist --tail 200 2>&1', {
      encoding: 'utf-8',
      cwd: path.join(__dirname, '../..'),
    });
  } catch (error: any) {
    return error.stdout || '';
  }
}

test.describe('Dynamic Configuration Reload', () => {
  let originalConfig: string;

  test.beforeAll(() => {
    // Save original configuration
    originalConfig = readConfig();
  });

  test.afterAll(() => {
    // Restore original configuration
    writeConfig(originalConfig);
  });

  test('should reload configuration when file changes', async () => {
    // Step 1: Get baseline logs
    const logsBefore = getContainerLogs();
    const reloadCountBefore = (logsBefore.match(/Configuration reloaded successfully/g) || []).length;

    // Step 2: Modify configuration (change service description to trigger reload)
    const updatedConfig = originalConfig.replace(
      'E2E verification environment with whitelist',
      'E2E verification environment with whitelist - MODIFIED'
    );
    writeConfig(updatedConfig);

    // Step 3: Wait for config watcher to detect and reload (fsnotify is fast, wait 2s to be safe)
    await new Promise(resolve => setTimeout(resolve, 2000));

    // Step 4: Check logs for reload message
    const logsAfter = getContainerLogs();
    const reloadCountAfter = (logsAfter.match(/Configuration reloaded successfully/g) || []).length;
    const hasReloadMessage = logsAfter.includes('Config content change detected, starting reload');

    // Verify that config reload was detected
    expect(hasReloadMessage).toBe(true);
    expect(reloadCountAfter).toBeGreaterThan(reloadCountBefore);

    // Step 5: Restore original config
    writeConfig(originalConfig);
    await new Promise(resolve => setTimeout(resolve, 2000));

    // Step 6: Verify another reload happened
    const logsFinal = getContainerLogs();
    const reloadCountFinal = (logsFinal.match(/Configuration reloaded successfully/g) || []).length;
    expect(reloadCountFinal).toBeGreaterThan(reloadCountAfter);
  });
});
