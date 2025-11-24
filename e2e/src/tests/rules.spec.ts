import { test, expect } from '@playwright/test'
import { routeStubAuthRequests } from '../support/stub-auth-route'

test.describe('Rules Configuration', () => {
  test.beforeEach(async ({ page }) => {
    await routeStubAuthRequests(page)
  })
  test('should allow access to /embed.js without authentication (prefix match)', async ({ page }) => {
    // Try to access /embed.js without authentication
    // This should work because /embed.js matches a rule with allow action
    const response = await page.goto('http://localhost:4183/embed.js')

    // Should not redirect to login
    expect(response?.status()).toBe(200)
    expect(page.url()).not.toContain('/_auth/login')

    // Should return JavaScript content
    const contentType = response?.headers()['content-type']
    expect(contentType).toContain('javascript')

    const content = await page.content()
    expect(content).toContain('ChatbotGate embed widget loaded')
  })

  test('should allow access to /public/* without authentication (prefix match)', async ({ page }) => {
    // Try to access /public/data.json without authentication
    const response = await page.goto('http://localhost:4183/public/data.json')

    // Should not redirect to login
    expect(response?.status()).toBe(200)
    expect(page.url()).not.toContain('/_auth/login')

    // Should return JSON content
    const contentType = response?.headers()['content-type']
    expect(contentType).toContain('json')

    const data = await response?.json()
    expect(data).toEqual({
      message: 'public data',
      status: 'ok',
    })
  })

  test('should allow access to /static/* without authentication (minimatch pattern)', async ({ page }) => {
    // Try to access /static/image.png without authentication
    const response = await page.goto('http://localhost:4183/static/image.png')

    // Should not redirect to login
    expect(response?.status()).toBe(200)
    expect(page.url()).not.toContain('/_auth/login')

    // Should return PNG content
    const contentType = response?.headers()['content-type']
    expect(contentType).toContain('image/png')
  })

  test('should allow access to /api/public/* without authentication (regex match)', async ({ page }) => {
    // Try to access /api/public/info without authentication
    const response = await page.goto('http://localhost:4183/api/public/info')

    // Should not redirect to login
    expect(response?.status()).toBe(200)
    expect(page.url()).not.toContain('/_auth/login')

    // Should return JSON content
    const data = await response?.json()
    expect(data).toHaveProperty('api', 'public')
    expect(data).toHaveProperty('version', '1.0')
    expect(data).toHaveProperty('authenticated', false)
  })

  test('should still require authentication for paths that do not match any allow rule', async ({ page }) => {
    // Try to access / without authentication
    // This should redirect to login since it doesn't match any allow rule
    await page.goto('http://localhost:4183/')

    // Should redirect to login
    await page.waitForURL(/\/_auth\/login/)
    expect(page.url()).toContain('/_auth/login')
  })

  test('should still require authentication for /api/private/*', async ({ page }) => {
    // Try to access /api/private/data without authentication
    // This should redirect to login because only /api/public/* has an allow rule
    await page.goto('http://localhost:4183/api/private/data')

    // Should redirect to login
    await page.waitForURL(/\/_auth\/login/)
    expect(page.url()).toContain('/_auth/login')
  })

  test('should deny access to /admin even without authentication (deny rule)', async ({ page }) => {
    // Try to access /admin without authentication
    // This should return 403 Forbidden (not redirect to login)
    const response = await page.goto('http://localhost:4183/admin', {
      failOnStatusCode: false
    })

    // CRITICAL: Should return 403 Forbidden, NOT redirect to login
    expect(response?.status()).toBe(403)
    expect(page.url()).not.toContain('/_auth/login')

    // Should show access denied message
    const mainContent = page.locator('main, [role="main"], body')
    await expect(mainContent).toContainText(/Access Denied|Forbidden|アクセスが拒否されました/i)
  })

  test('should deny access to /admin even for authenticated users (deny rule)', async ({ page }) => {
    // First, authenticate via OAuth2
    await page.goto('http://localhost:4183/')
    await page.waitForURL(/\/_auth\/login/)

    // Authenticate (assuming stub-auth is available)
    await page.getByRole('link', { name: 'stub-auth' }).click()
    await expect(page).toHaveURL(/localhost:3001\/login/)

    await page.locator('[data-test="login-email"]').fill('someone@example.com')
    await page.locator('[data-test="login-password"]').fill('password')

    await Promise.all([
      page.waitForURL(/localhost:3001\/oauth\/authorize/),
      page.locator('[data-test="login-submit"]').click(),
    ])

    await Promise.all([
      page.waitForURL(/localhost:4183/),
      page.locator('[data-test="authorize-allow"]').click(),
    ])

    // Wait for page to load completely
    await page.waitForLoadState('networkidle')

    // Verify we're authenticated
    await expect(page.locator('[data-test="auth-status"]')).toContainText('true')

    // Now try to access /admin as authenticated user
    const response = await page.goto('http://localhost:4183/admin', {
      failOnStatusCode: false
    })

    // CRITICAL: Even authenticated users should be denied (403)
    expect(response?.status()).toBe(403)

    // Should show access denied message
    const mainContent = page.locator('main, [role="main"], body')
    await expect(mainContent).toContainText(/Access Denied|Forbidden|アクセスが拒否されました/i)
  })
})
