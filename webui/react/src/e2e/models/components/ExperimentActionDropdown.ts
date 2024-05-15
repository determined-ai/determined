import { DropdownContent } from 'e2e/models/hew/Dropdown';

/**
 * Returns the representation of the ActionDropdown menu defined by the User Admin page.
 * This constructor represents the InteractiveTable in src/components/ExperimentActionDropdown.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this ExperimentActionDropdown
 * @param {string} obj.selector - Used as a selector uesd to locate this object
 */
export class ExperimentActionDropdown extends DropdownContent {
  // TODO where is this thing? <Button icon={<Icon name="overflow-vertical" size="small" title="Action menu" />} />
  // TODO I'm assuming it's new tab, new window, copy value, etc
}
