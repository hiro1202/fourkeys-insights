import { test, expect } from '@playwright/test'

test.describe('Setup Flow', () => {
  test('displays setup page with step 1', async ({ page }) => {
    await page.goto('/')

    // Should show setup title
    await expect(page.locator('h2')).toContainText(/Setup|セットアップ/)

    // Should show step indicators (1, 2, 3)
    const stepIndicators = page.locator('.rounded-full')
    await expect(stepIndicators).toHaveCount(3)

    // Step 1 should be active (blue)
    const step1 = stepIndicators.first()
    await expect(step1).toHaveText('1')

    // Should show PAT label
    await expect(page.locator('text=GitHub Personal Access Token')).toBeVisible()

    // Should show validate button
    await expect(page.getByRole('button', { name: /Validate|検証/ })).toBeVisible()
  })

  test('validate button shows error for missing token', async ({ page }) => {
    await page.goto('/')

    // Click validate without token (API not running, should show error)
    const validateBtn = page.getByRole('button', { name: /Validate|検証/ })
    await validateBtn.click()

    // Should show an error (network error or invalid token)
    await expect(page.locator('text=/Invalid|無効|error/i')).toBeVisible({ timeout: 5000 })
  })

  test('language toggle switches to Japanese', async ({ page }) => {
    await page.goto('/')

    // Select Japanese
    const langSelect = page.locator('select').last()
    await langSelect.selectOption('ja')

    // Should show Japanese text
    await expect(page.locator('h2')).toContainText('セットアップ')
  })

  test('dark mode toggle works', async ({ page }) => {
    await page.goto('/')

    // Click dark mode toggle
    const darkBtn = page.getByLabel('Toggle dark mode')
    await darkBtn.click()

    // html should have 'dark' class
    const html = page.locator('html')
    await expect(html).toHaveClass(/dark/)

    // Toggle back
    await darkBtn.click()
    await expect(html).not.toHaveClass(/dark/)
  })
})

test.describe('Dashboard', () => {
  test('shows empty state when no groups exist', async ({ page }) => {
    await page.goto('/dashboard')

    // Should show no data message or empty state
    await expect(page.locator('text=/No data|データがありません/i')).toBeVisible()
  })
})
