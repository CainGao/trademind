// Setup 向导相关 API。

import client from "./client";

export interface SetupStatus {
  completed: boolean;
  steps: string[];
  current_step: string;
  scenario: string;
  company_configured: boolean;
  ai_key_configured: boolean;
  password_changed: boolean;
}

export interface CompanyInput {
  name: string;
  industry?: string;
  country?: string;
  contact?: string;
}

export interface ScenarioInput {
  scenario: "b2b" | "b2c" | "both";
}

export interface AIKeyInput {
  deepseek_key?: string;
  openai_key?: string;
  qwen_key?: string;
  anthropic_key?: string;
  default_model?: "deepseek" | "openai" | "qwen" | "anthropic";
}

export interface ChangePasswordInput {
  old_password: string;
  new_password: string;
}

export const setupApi = {
  status: () => client.get<unknown, SetupStatus>("/setup/status"),

  saveCompany: (input: CompanyInput) =>
    client.post<unknown, { ok: boolean }>("/setup/company", input),

  selectScenario: (input: ScenarioInput) =>
    client.post<unknown, { ok: boolean; scenario: string }>("/setup/scenario", input),

  saveAIKeys: (input: AIKeyInput) =>
    client.post<unknown, { ok: boolean }>("/setup/ai-key", input),

  changePassword: (input: ChangePasswordInput) =>
    client.post<unknown, { ok: boolean }>("/setup/change-password", input),

  complete: () =>
    client.post<unknown, { ok: boolean; completed: boolean }>("/setup/complete"),
};
