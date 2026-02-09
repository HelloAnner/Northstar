export type RuleSeverity = 'info' | 'warn' | 'error'

export interface IndicatorDefinition {
  code: string
  name: string
  groupCode: string
  groupName: string
  groupOrder: number
  description: string
  formula: string
  unit: string
  floatMin: number
  floatMax: number
  displayOrder: number
  enabled: boolean
}

export interface RuleIndicatorLink {
  ruleCode: string
  indicatorCode: string
  relationLabel: string
  weight: number
  displayOrder: number
}

export interface RuleDefinition {
  ruleCode: string
  name: string
  description: string
  expression: string
  severity: RuleSeverity
  suggestion: string
  preferenceJson: string
  displayOrder: number
  enabled: boolean
}

export interface RuleDetail {
  rule: RuleDefinition
  links: RuleIndicatorLink[]
}


export type RuleEvaluationStatus = 'pass' | 'fail' | 'skipped'

export interface RuleEvaluationIndicator {
  indicatorCode: string
  indicatorName: string
  relationLabel?: string
  weight?: number
  value: number
  threshold?: number
}

export interface RuleEvaluation {
  ruleCode: string
  ruleName: string
  description: string
  severity: RuleSeverity
  expression: string
  status: RuleEvaluationStatus
  message: string
  suggestion?: string
  failedCount: number
  failedIndicators: RuleEvaluationIndicator[]
  skippedReason?: string
  evaluatedBindings: number
}

export interface UpsertIndicatorDefinitionPayload {
  name: string
  groupCode: string
  groupName: string
  groupOrder: number
  description: string
  formula: string
  unit: string
  floatMin: number
  floatMax: number
  displayOrder: number
  enabled: boolean
}

export interface UpsertRulePayload {
  name: string
  description: string
  expression: string
  severity: RuleSeverity
  suggestion: string
  preferenceJson: string
  displayOrder: number
  enabled: boolean
  links: RuleIndicatorLink[]
}
