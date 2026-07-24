export interface Binding {
  id: string;
  name: string;
  key_hash: string;
  key_preview: string;
  allow: string[];
  enabled: boolean;
}

// A selectable provider/account shown in the multi-select.
// value is what gets stored in Binding.allow:
//   - "auth:<id>" pins a specific credential (from auth files)
//   - a bare provider name (e.g. "claude", "openrouter") covers all accounts of that provider
export interface ProviderOption {
  value: string;
  label: string;
  kind: "auth" | "provider";
  meta?: string;
}
