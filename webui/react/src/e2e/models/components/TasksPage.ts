import { NamedComponent } from 'e2e/models/BaseComponent';

/**
 * Represents the TasksComponent in the TasksComponent component
 */
export class TasksComponent extends NamedComponent {
  override defaultSelector: string = '[id$=tasks]';
}
