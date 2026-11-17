/** @param {import("@playwright/test").Page} page */
export async function openOrganizationSwitcher(page) {
  const trigger = page.getByRole("button", { name: "我的账户", exact: true });
  const panel = page.getByRole("dialog", { name: "账户快捷操作", exact: true });
  if (!(await panel.isVisible())) await trigger.click();
  return panel.getByRole("combobox", { name: "切换企业", exact: true });
}

/** @param {import("@playwright/test").Page} page @param {string} id */
export async function selectWorkbenchOrganization(page, id) {
  await (await openOrganizationSwitcher(page)).selectOption(id);
  await page.keyboard.press("Escape");
}
