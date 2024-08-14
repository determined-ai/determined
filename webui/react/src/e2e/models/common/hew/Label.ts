import { NamedComponent } from 'playwright-page-model-base/BaseComponent';

/**
 * Represents the Toggle component in hew/src/kit/Toggle.tsx
 */
export class Label extends NamedComponent {
  override defaultSelector = '[class^="Label_base"]';
}
