import { pageSource, validateCapture } from './validation';
import { amazonPageSource, validateAmazonCapture } from './amazon/validation';

// Two explicit approved sources, with each source owner's strict validator.
export function capturePageSource(input: string) {
  if (new URL(input).hostname.toLowerCase() === 'detail.1688.com') return { platform:'1688' as const, ...pageSource(input) };
  return { platform:'amazon' as const, ...amazonPageSource(input) };
}
export function validatePlatformCapture(value: unknown) {
  if (value && typeof value === 'object' && !Array.isArray(value) && 'source' in value && value.source === 'amazon') return validateAmazonCapture(value);
  return validateCapture(value);
}
