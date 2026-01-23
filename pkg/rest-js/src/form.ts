import { z } from "zod";

export const LangSchema = z.enum(["en", "fr"]);
export type Lang = z.infer<typeof LangSchema>;

export const SchemaSourceSchema = z.enum(["USER", "AI", "FORK", "EXTERNAL"]);
export type SchemaSource = z.infer<typeof SchemaSourceSchema>;

export const ModuleIDSchema = z.string().regex(/^[a-z0-9]+(-[a-z0-9]+)*$/);
export type ModuleID = z.infer<typeof ModuleIDSchema>;

export const ModuleNamespaceSchema = z.string().regex(/^[a-z0-9]+(-[a-z0-9]+)*$/);
export type ModuleNamespace = z.infer<typeof ModuleNamespaceSchema>;

export const ModuleVersionSchema = z.string().regex(/^[0-9]+\.[0-9]+\.[0-9]+$/);
export type ModuleVersion = z.infer<typeof ModuleVersionSchema>;

// Matches Go patterns from internal/lib/moduleString.go
const ModuleNameRegex = "[a-z0-9]+(-[a-z0-9]+)*";
const ModuleVersionRegex = "[0-9]+\\.[0-9]+\\.[0-9]+";
const ModulePreversionRegex = "(-[a-z0-9]+)*";

export const ModulePreversionSchema = z
  .string()
  .regex(new RegExp(`^${ModulePreversionRegex}$`))
  .optional();
export type ModulePreversion = z.infer<typeof ModulePreversionSchema>;

export const ModuleStringSchema = z
  .string()
  .regex(new RegExp(`^${ModuleNameRegex}:${ModuleNameRegex}@v${ModuleVersionRegex}${ModulePreversionRegex}$`));
export type ModuleString = z.infer<typeof ModuleStringSchema>;

export const UUIDSchema = z.uuid();
export type UUID = z.infer<typeof UUIDSchema>;

export const LimitSchema = z.int().min(1).max(100).optional();
export type Limit = z.infer<typeof LimitSchema>;

export const OffsetSchema = z.int().min(0).optional();
export type Offset = z.infer<typeof OffsetSchema>;
