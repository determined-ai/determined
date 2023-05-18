import { ValueOf } from 'shared/types';

export const Status = {
  Active: 'var(--theme-status-active)',
  ActiveOn: 'var(--theme-status-active-on)',
  ActiveOnStrong: 'var(--theme-status-active-on-strong)',
  ActiveOnWeak: 'var(--theme-status-active-on-weak)',
  ActiveStrong: 'var(--theme-status-active-strong)',
  ActiveWeak: 'var(--theme-status-active-weak)',
  Critical: 'var(--theme-status-critical)',
  CriticalOn: 'var(--theme-status-critical-on)',
  CriticalOnStrong: 'var(--theme-status-critical-on-strong)',
  CriticalOnWeak: 'var(--theme-status-critical-on-weak)',
  CriticalStrong: 'var(--theme-status-critical-strong)',
  CriticalWeak: 'var(--theme-status-critical-weak)',
  Inactive: 'var(--theme-status-inactive)',
  InactiveOn: 'var(--theme-status-inactive-on)',
  InactiveOnStrong: 'var(--theme-status-inactive-on-strong)',
  InactiveOnWeak: 'var(--theme-status-inactive-on-weak)',
  InactiveStrong: 'var(--theme-status-inactive-strong)',
  InactiveWeak: 'var(--theme-status-inactive-weak)',
  Pending: 'var(--theme-status-pending)',
  PendingOn: 'var(--theme-status-pending-on)',
  PendingOnStrong: 'var(--theme-status-pending-on-strong)',
  PendingOnWeak: 'var(--theme-status-pending-on-weak)',
  PendingStrong: 'var(--theme-status-pending-strong)',
  PendingWeak: 'var(--theme-status-pending-weak)',
  Potential: 'var(--theme-status-potential)',
  Success: 'var(--theme-status-success)',
  SuccessOn: 'var(--theme-status-success-on)',
  SuccessOnStrong: 'var(--theme-status-success-on-strong)',
  SuccessOnWeak: 'var(--theme-status-success-on-weak)',
  SuccessStrong: 'var(--theme-status-success-strong)',
  SuccessWeak: 'var(--theme-status-success-weak)',
  Warning: 'var(--theme-status-warning)',
  WarningOn: 'var(--theme-status-warning-on)',
  WarningOnStrong: 'var(--theme-status-warning-on-strong)',
  WarningOnWeak: 'var(--theme-status-warning-on-weak)',
  WarningStrong: 'var(--theme-status-warning-strong)',
  WarningWeak: 'var(--theme-status-warning-weak)',
} as const;

export const Background = {
  Background: 'var(--theme-background)',
  BackgroundBorder: 'var(--theme-background-border)',
  BackgroundBorderStrong: 'var(--theme-background-border-strong)',
  BackgroundBorderWeak: 'var(--theme-background-border-weak)',
  BackgroundOn: 'var(--theme-background-on)',
  BackgroundOnStrong: 'var(--theme-background-on-strong)',
  BackgroundOnWeak: 'var(--theme-background-on-weak)',
  BackgroundStrong: 'var(--theme-background-strong)',
  BackgroundWeak: 'var(--theme-background-weak)',
} as const;

export const Stage = {
  Stage: 'var(--theme-stage)',
  StageBorder: 'var(--theme-stage-border)',
  StageBorderStrong: 'var(--theme-stage-border-strong)',
  StageBorderWeak: 'var(--theme-stage-border-weak)',
  StageOn: 'var(--theme-stage-on)',
  StageOnStrong: 'var(--theme-stage-on-strong)',
  StageOnWeak: 'var(--theme-stage-on-weak)',
  StageStrong: 'var(--theme-stage-strong)',
  StageWeak: 'var(--theme-stage-weak)',
} as const;

export const Surface = {
  Surface: 'var(--theme-surface)',
  SurfaceBorder: 'var(--theme-surface-border)',
  SurfaceBorderStrong: 'var(--theme-surface-border-strong)',
  SurfaceBorderWeak: 'var(--theme-surface-border-weak)',
  SurfaceOn: 'var(--theme-surface-on)',
  SurfaceOnStrong: 'var(--theme-surface-on-strong)',
  SurfaceOnWeak: 'var(--theme-surface-on-weak)',
  SurfaceStrong: 'var(--theme-surface-strong)',
  SurfaceWeak: 'var(--theme-surface-weak)',
} as const;

export const Float = {
  Float: 'var(--theme-float)',
  FloatBorder: 'var(--theme-float-border)',
  FloatBorderStrong: 'var(--theme-float-border-strong)',
  FloatBorderWeak: 'var(--theme-float-border-weak)',
  FloatOn: 'var(--theme-float-on)',
  FloatOnStrong: 'var(--theme-float-on-strong)',
  FloatOnWeak: 'var(--theme-float-on-weak)',
  FloatStrong: 'var(--theme-float-strong)',
  FloatWeak: 'var(--theme-float-weak)',
} as const;

export const Overlay = {
  Overlay: 'var(--theme-overlay)',
  OverlayStrong: 'var(--theme-overlay-strong)',
  OverlayWeak: 'var(--theme-overlay-weak)',
} as const;

export const Brand = {
  Brand: 'var(--theme-brand)',
  BrandStrong: 'var(--theme-brand-strong)',
  BrandWeak: 'var(--theme-brand-weak)',
} as const;

export const Interactive = {
  Interactive: 'var(--theme-ix)',
  InteractiveActive: 'var(--theme-ix-active)',
  InteractiveBorder: 'var(--theme-ix-border)',
  InteractiveBorderActive: 'var(--theme-ix-border-active)',
  InteractiveBorderInactive: 'var(--theme-ix-border-inactive)',
  InteractiveBorderStrong: 'var(--theme-ix-border-strong)',
  InteractiveBorderWeak: 'var(--theme-ix-border-weak)',
  InteractiveInactive: 'var(--theme-ix-inactive)',
  InteractiveOn: 'var(--theme-ix-on)',
  InteractiveOnActive: 'var(--theme-ix-on-active)',
  InteractiveOnInactive: 'var(--theme-ix-on-inactive)',
  InteractiveOnStrong: 'var(--theme-ix-on-strong)',
  InteractiveOnWeak: 'var(--theme-ix-on-weak)',
  InteractiveStrong: 'var(--theme-ix-strong)',
  InteractiveWeak: 'var(--theme-ix-weak)',
} as const;

export type Status = ValueOf<typeof Status>;
export type Background = ValueOf<typeof Background>;
export type Stage = ValueOf<typeof Stage>;
export type Surface = ValueOf<typeof Surface>;
export type Float = ValueOf<typeof Float>;
export type Overlay = ValueOf<typeof Overlay>;
export type Brand = ValueOf<typeof Brand>;
export type Interactive = ValueOf<typeof Interactive>;
