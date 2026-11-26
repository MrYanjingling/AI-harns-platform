export const ASSISTANT_SESSION_KEY = 'gateway.ai.assistant.session'
export const ASSISTANT_ATTACHMENTS_KEY = 'gateway.ai.assistant.attachments'

export function clearAssistantSession(): void {
  try {
    window.sessionStorage.removeItem(ASSISTANT_SESSION_KEY)
    window.sessionStorage.removeItem(ASSISTANT_ATTACHMENTS_KEY)
  } catch {
    // Ignore storage failures.
  }
}
