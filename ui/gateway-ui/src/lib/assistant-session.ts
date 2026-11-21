export const ASSISTANT_SESSION_KEY = 'gateway.ai.assistant.session'

export function clearAssistantSession(): void {
  try {
    window.sessionStorage.removeItem(ASSISTANT_SESSION_KEY)
  } catch {
    // Ignore storage failures.
  }
}
