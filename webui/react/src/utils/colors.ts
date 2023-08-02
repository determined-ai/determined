/* eslint-disable sort-keys-fix/sort-keys-fix */
import { ValueOf } from 'types';

export const Status = {
  Active: 'var(--theme-status-active)',
  ActiveStrong: 'var(--theme-status-active-strong)',
  ActiveWeak: 'var(--theme-status-active-weak)',
  ActiveOn: 'var(--theme-status-active-on)',
  ActiveOnStrong: 'var(--theme-status-active-on-strong)',
  ActiveOnWeak: 'var(--theme-status-active-on-weak)',
  Critical: 'var(--theme-status-critical)',
  CriticalStrong: 'var(--theme-status-critical-strong)',
  CriticalWeak: 'var(--theme-status-critical-weak)',
  CriticalOn: 'var(--theme-status-critical-on)',
  CriticalOnStrong: 'var(--theme-status-critical-on-strong)',
  CriticalOnWeak: 'var(--theme-status-critical-on-weak)',
  Inactive: 'var(--theme-status-inactive)',
  InactiveStrong: 'var(--theme-status-inactive-strong)',
  InactiveWeak: 'var(--theme-status-inactive-weak)',
  InactiveOn: 'var(--theme-status-inactive-on)',
  InactiveOnStrong: 'var(--theme-status-inactive-on-strong)',
  InactiveOnWeak: 'var(--theme-status-inactive-on-weak)',
  Pending: 'var(--theme-status-pending)',
  PendingStrong: 'var(--theme-status-pending-strong)',
  PendingWeak: 'var(--theme-status-pending-weak)',
  PendingOn: 'var(--theme-status-pending-on)',
  PendingOnStrong: 'var(--theme-status-pending-on-strong)',
  PendingOnWeak: 'var(--theme-status-pending-on-weak)',
  Success: 'var(--theme-status-success)',
  SuccessStrong: 'var(--theme-status-success-strong)',
  SuccessWeak: 'var(--theme-status-success-weak)',
  SuccessOn: 'var(--theme-status-success-on)',
  SuccessOnStrong: 'var(--theme-status-success-on-strong)',
  SuccessOnWeak: 'var(--theme-status-success-on-weak)',
  Warning: 'var(--theme-status-warning)',
  WarningStrong: 'var(--theme-status-warning-strong)',
  WarningWeak: 'var(--theme-status-warning-weak)',
  WarningOn: 'var(--theme-status-warning-on)',
  WarningOnStrong: 'var(--theme-status-warning-on-strong)',
  WarningOnWeak: 'var(--theme-status-warning-on-weak)',
  Potential: 'var(--theme-status-potential)',
} as const;

export const Background = {
  Background: 'var(--theme-background)',
  BackgroundStrong: 'var(--theme-background-strong)',
  BackgroundWeak: 'var(--theme-background-weak)',
  BackgroundOn: 'var(--theme-background-on)',
  BackgroundOnStrong: 'var(--theme-background-on-strong)',
  BackgroundOnWeak: 'var(--theme-background-on-weak)',
  BackgroundBorder: 'var(--theme-background-border)',
  BackgroundBorderStrong: 'var(--theme-background-border-strong)',
  BackgroundBorderWeak: 'var(--theme-background-border-weak)',
} as const;

export const Stage = {
  Stage: 'var(--theme-stage)',
  StageStrong: 'var(--theme-stage-strong)',
  StageWeak: 'var(--theme-stage-weak)',
  StageBorder: 'var(--theme-stage-border)',
  StageBorderStrong: 'var(--theme-stage-border-strong)',
  StageBorderWeak: 'var(--theme-stage-border-weak)',
  StageOn: 'var(--theme-stage-on)',
  StageOnStrong: 'var(--theme-stage-on-strong)',
  StageOnWeak: 'var(--theme-stage-on-weak)',
} as const;

export const Surface = {
  Surface: 'var(--theme-surface)',
  SurfaceStrong: 'var(--theme-surface-strong)',
  SurfaceWeak: 'var(--theme-surface-weak)',
  SurfaceBorder: 'var(--theme-surface-border)',
  SurfaceBorderStrong: 'var(--theme-surface-border-strong)',
  SurfaceBorderWeak: 'var(--theme-surface-border-weak)',
  SurfaceOn: 'var(--theme-surface-on)',
  SurfaceOnStrong: 'var(--theme-surface-on-strong)',
  SurfaceOnWeak: 'var(--theme-surface-on-weak)',
} as const;

export const Float = {
  Float: 'var(--theme-float)',
  FloatStrong: 'var(--theme-float-strong)',
  FloatWeak: 'var(--theme-float-weak)',
  FloatBorder: 'var(--theme-float-border)',
  FloatBorderStrong: 'var(--theme-float-border-strong)',
  FloatBorderWeak: 'var(--theme-float-border-weak)',
  FloatOn: 'var(--theme-float-on)',
  FloatOnStrong: 'var(--theme-float-on-strong)',
  FloatOnWeak: 'var(--theme-float-on-weak)',
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
  InteractiveStrong: 'var(--theme-ix-strong)',
  InteractiveWeak: 'var(--theme-ix-weak)',
  InteractiveActive: 'var(--theme-ix-active)',
  InteractiveInactive: 'var(--theme-ix-inactive)',
  InteractiveBorder: 'var(--theme-ix-border)',
  InteractiveBorderStrong: 'var(--theme-ix-border-strong)',
  InteractiveBorderWeak: 'var(--theme-ix-border-weak)',
  InteractiveBorderActive: 'var(--theme-ix-border-active)',
  InteractiveBorderInactive: 'var(--theme-ix-border-inactive)',
  InteractiveOn: 'var(--theme-ix-on)',
  InteractiveOnStrong: 'var(--theme-ix-on-strong)',
  InteractiveOnWeak: 'var(--theme-ix-on-weak)',
  InteractiveOnActive: 'var(--theme-ix-on-active)',
  InteractiveOnInactive: 'var(--theme-ix-on-inactive)',
} as const;

export type Status = ValueOf<typeof Status>;
export type Background = ValueOf<typeof Background>;
export type Stage = ValueOf<typeof Stage>;
export type Surface = ValueOf<typeof Surface>;
export type Float = ValueOf<typeof Float>;
export type Overlay = ValueOf<typeof Overlay>;
export type Brand = ValueOf<typeof Brand>;
export type Interactive = ValueOf<typeof Interactive>;
