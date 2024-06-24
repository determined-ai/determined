import { NamedComponent } from 'e2e/models/common/base/BaseComponent';

/**
 * Represents the TasksComponent in the TasksComponent component
 */
export class TasksComponent extends NamedComponent {
  override defaultSelector: string = '[id$=tasks]';
}
