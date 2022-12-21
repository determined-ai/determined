import { PoweroffOutlined } from '@ant-design/icons';
import {
  Card,
  Input, //TODO: Replace with kit component
  InputNumber, //TODO: Replace with kit component
  Space,
} from 'antd';
import React, { useEffect } from 'react';
import { Link } from 'react-router-dom';

import OverviewStats from 'components/OverviewStats'; //TODO: Rename?
import ResourcePoolCard from 'components/ResourcePoolCard'; //TODO: Rename?
import UserAvatar from 'components/UserAvatar'; //TODO: Rename?
import ExperimentDetailsHeader from './ExperimentDetails/ExperimentDetailsHeader'; //TODO: Rename?

import Pivot from 'components/kit/Pivot';
import Tooltip from 'components/kit/Tooltip';
import Pagination from 'components/kit/Pagination';
import Button from 'components/kit/Button';
import Breadcrumb from 'components/kit/Breadcrumb';
import Checkbox from 'components/kit/Checkbox';

import LogViewer from 'components/LogViewer/LogViewer';  //TODO: Move to components/kit?
import SelectFilter from 'components/SelectFilter'; //TODO: Move to components/kit?
import ResponsiveTable from 'components/Table/ResponsiveTable'; //TODO: Move to components/kit?

import Grid, { GridMode } from 'components/Grid'; //TODO: Move to components/kit? Add section to DesignKit page?
import Logo from 'components/Logo'; //TODO: Move to components/kit? Add section to DesignKit page?
import Page from 'components/Page'; //TODO: Move to components/kit? Add section to DesignKit page?
import ThemeToggle from 'components/ThemeToggle'; //TODO: Move to components/kit? Add section to DesignKit page?

import resourcePools from 'fixtures/responses/cluster/resource-pools.json';
import { V1LogLevel } from 'services/api-ts-sdk';
import { mapV1LogsResponse } from 'services/decoder';
import Icon from 'shared/components/Icon';
import useUI from 'shared/contexts/stores/UI';
import { ValueOf } from 'shared/types';
import { generateTestExperimentData } from 'storybook/shared/generateTestData';
import { ShirtSize } from 'themes';
import { BrandingType, ResourcePool } from 'types';

import css from './DesignKit.module.scss';

const Components = {
  ActionBar: 'ActionBar',
  Breadcrumbs: 'Breadcrumbs',
  Buttons: 'Buttons',
  Checkboxes: 'Checkboxes',
  DataCards: 'DataCards',
  Dropdowns: 'Comboboxes & Dropdowns',
  Facepile: 'Facepile',
  Lists: 'Lists (tables)',
  LogViewer: 'LogViewer',
  Pagination: 'Pagination',
  Pivot: 'Pivot',
  Searchboxes: 'Searchboxes',
  Spinbuttons: 'Spinbuttons',
  Textfields: 'Input - Textfields',
  Tooltips: 'Tooltips',
} as const;

type ComponentNames = ValueOf<typeof Components>;
type ComponentIds = keyof typeof Components;

const componentOrder: ComponentIds[] = [
  'Buttons',
  'Dropdowns',
  'Checkboxes',
  'Searchboxes',
  'Spinbuttons',
  'Textfields',
  'Lists',
  'Breadcrumbs',
  'Facepile',
  'ActionBar',
  'Pivot',
  'Pagination',
  'DataCards',
  'LogViewer',
  'Tooltips',
];

interface Props {
  children?: React.ReactNode;
  id: ComponentIds;
  title: ComponentNames;
}

const ComponentSection: React.FC<Props> = ({ children, id, title }: Props): JSX.Element => {
  return (
    <section>
      <h3 id={id}>{title}</h3>
      {children}
    </section>
  );
};

const ButtonsSection: React.FC = () => {
  return (
    <ComponentSection id="Buttons" title="Buttons">
      <Card>
        <p>
          <code>{'<Button>'}</code>s give people a way to trigger an action. They&apos;re typically found in forms,
          dialog panels, and dialogs. Some buttons are specialized for particular tasks, such as
          navigation, repeated actions, or presenting menus.
        </p>
      </Card>
      <Card title="Best practices">
        <strong>Layout</strong>
        <ul>
          <li>
            For dialog boxes and panels, where people are moving through a sequence of screens,
            right-align buttons with the container.
          </li>
          <li>For single-page forms and focused tasks, left-align buttons with the container.</li>
          <li>
            Always place the primary button on the left, the secondary button just to the right of
            it.
          </li>
          <li>
            Show only one primary button that inherits theme color at rest state. If there are more
            than two buttons with equal priority, all buttons should have neutral backgrounds.
          </li>
          <li>
            Don&apos;t use a button to navigate to another place; use a link instead. The exception
            is in a wizard where &quot;Back&quot; and &quot;Next&quot; buttons may be used.
          </li>
          <li>
            Don&apos;t place the default focus on a button that destroys data. Instead, place the
            default focus on the button that performs the &quot;safe act&quot; and retains the
            content (such as &quot;Save&quot;) or cancels the action (such as &quot;Cancel&quot;).
          </li>
        </ul>
        <strong>Content</strong>
        <ul>
          <li>Use sentence-style capitalization—only capitalize the first word.</li>
          <li>
            Make sure it&apos;s clear what will happen when people interact with the button. Be
            concise; usually a single verb is best. Include a noun if there is any room for
            interpretation about what the verb means. For example, &quot;Delete folder&quot; or
            &quot;Create account&quot;.
          </li>
        </ul>
        <strong>Accessibility</strong>
        <ul>
          <li>Always enable the user to navigate to focus on buttons using their keyboard.</li>
          <li>Buttons need to have accessible naming.</li>
          <li>Aria- and roles need to have consistent (non-generic) attributes.</li>
        </ul>
      </Card>
      <Card title="Usage">
        <strong>Default Button</strong>
        <Space>
          <Button type="primary">Primary</Button>
          <Button>Secondary</Button>
          <Button loading>Loading</Button>
          <Button disabled>Disabled</Button>
        </Space>
        <strong>Guiding principles</strong>
        <ul>
          <li>15px inner horizontal padding</li>
          <li>8px inner vertical padding</li>
          <li>8px external margins</li>
          <li className={css.warning}>Colors do not meet accessibility guidelines</li>
        </ul>
        <hr />
        <strong>Default Button with icon</strong>
        <Space>
          <Button icon={<PoweroffOutlined />} type="primary">
            ButtonWithIcon
          </Button>
          <Button icon={<PoweroffOutlined />}>ButtonWithIcon</Button>
          <Button disabled icon={<PoweroffOutlined />}>
            ButtonWithIcon
          </Button>
        </Space>
        <strong>Guiding principles</strong>
        <ul>
          <li>15px inner horizontal padding</li>
          <li>8px inner vertical padding</li>
          <li>8px padding between icon and text</li>
          <li>8px external margins</li>
          <li className={css.warning}>Colors do not meet accessibility guidelines</li>
        </ul>
        <hr />
        <strong>Large iconic buttons</strong>
        <Space>
          <Button
            style={{
              height: '100%',
              padding: '16px',
              paddingBottom: '8px',
              width: '120px',
            }}
            type="primary">
            <div style={{ alignItems: 'center', display: 'flex', flexDirection: 'column' }}>
              <Icon name="searcher-grid" />
              <p>Iconic button</p>
            </div>
          </Button>
          <Button
            style={{
              height: '100%',
              padding: '16px',
              paddingBottom: '8px',
              width: '120px',
            }}>
            <div style={{ alignItems: 'center', display: 'flex', flexDirection: 'column' }}>
              <Icon name="searcher-grid" />
              <p>Iconic button</p>
            </div>
          </Button>
          <Button
            disabled
            style={{
              height: '100%',
              padding: '16px',
              paddingBottom: '8px',
              width: '120px',
            }}>
            <div style={{ alignItems: 'center', display: 'flex', flexDirection: 'column' }}>
              <Icon name="searcher-grid" />
              <p>Iconic button</p>
            </div>
          </Button>
        </Space>
        <strong>Guiding principles</strong>
        <ul>
          <li>Component needs to be reviewed/looked at.</li>
          <li>Missing distinguishing states</li>
          <li>Visual density</li>
        </ul>
      </Card>
    </ComponentSection>
  );
};

const DropdownsSection: React.FC = () => {
  return (
    <ComponentSection id="Dropdowns" title="Comboboxes & Dropdowns">
      <Card>
        <p>
          A dropdown/combo box (<code>{'<SelectFilter>'}</code>) combines a text field and a dropdown giving people a way to select an
          option from a list or enter their own choice.
        </p>
      </Card>
      <Card title="Best practices">
        <strong>Layout</strong>
        <ul>
          <li>
            Use a combo box when there are multiple choices that can be collapsed under one title,
            when the list of items is long, or when space is constrained.
          </li>
        </ul>
        <strong>Content</strong>
        <ul>
          <li>Use single words or shortened statements as options.</li>
          <li>Don&apos;t use punctuation at the end of options.</li>
        </ul>
        <strong>Accessibility</strong>
        <ul>
          <li>
            ComboBox dropdowns render in their own layer by default to ensure they are not clipped
            by containers with overflow: hidden or overflow: scroll. This causes extra difficulty
            for people who use screen readers, so we recommend rendering the ComboBox options
            dropdown inline unless they are in overflow containers.
          </li>
        </ul>
        <strong>Truncation</strong>
        <ul>
          <li>
            By default, the ComboBox truncates option text instead of wrapping to a new line.
            Because this can lose meaningful information, it is recommended to adjust styles to wrap
            the option text.
          </li>
        </ul>
      </Card>
      <Card title="Usage">
        <strong>Basic dropdown with inline options</strong>
        <Space>
          <SelectFilter
            defaultValue={1}
            options={[
              { label: 'Option 1', value: 1 },
              { label: 'Option 2', value: 2 },
              { label: 'Option 3', value: 3 },
            ]}
          />
          <SelectFilter
            defaultValue="disabled"
            disabled
            options={[{ label: 'Disabled', value: 'disabled' }]}
          />
        </Space>
        <strong>Guiding principles</strong>
        <ul>
          <li>16px inner horizontal padding</li> <li>5px inner vertical padding</li>
          <li>8px minimum inner horizontal padding</li> <li>8px external margins</li>
          <li>4px for the start of the option items</li>
          <li className={css.warning}>Colors do not meet accessibility guidelines</li>
        </ul>
        <hr />
        <strong>Dropdown menu items</strong>
        <strong>Guiding principles</strong>
        <ul>
          <li>Needs to be same width as dropdown box</li>
          <li>Top item has rounded top corners</li>
          <li>Middle items have no rounded corners</li>
          <li>Bottom item has rounded bottom corners</li>
          <li>12px inner horizontal padding</li>
          <li>5px inner vertical padding</li>
          <li>8px minimum inner horizontal padding</li>
          <li>8px external margins</li>
          <li className={css.warning}>Colors do not meet accessibility guidelines</li>
        </ul>
        <strong>Menu items with checkmark</strong>
        <p>Not implemented</p>
        <strong>Guiding principles</strong>
        <ul>
          <li>Needs to be same width as dropdown box</li>
          <li>Preserve 12px right padding to checkmark</li>
          <li>5px inner vertical padding</li>
          <li>8px minimum inner horizontal padding</li>
          <li>8px external margins</li>
          <li className={css.warning}>Colors do not meet accessibility guidelines</li>
        </ul>
        <hr />
        <strong>Categorical menu items</strong>
        <strong>Guiding principles</strong>
        <ul>
          <li>A parent category can never be without children.</li>
          <li>
            The parent cell has visually distinctive features to make it stand out in a multi-parent
            list from children.
          </li>
          <li>Same padding as normal cells</li>
          <li className={css.warning}>Colors do not meet accessibility guidelines</li>
        </ul>
        <hr />
        <strong>Categories with checkmarks</strong>
        <p>Not implemented</p>
        <strong>Guiding principles</strong>
        <ul>
          <li>The padding of the children is preserved and aligns nicely.</li>
        </ul>
        <hr />
        <strong>Categories with checkmarks and icons</strong>
        <p>Not implemented</p>
        <strong>Guiding principles</strong>
        <ul>
          <li>The padding of the children is preserved and aligns nicely.</li>
        </ul>
        <hr />
        <strong>Selecting all categories</strong>
        <strong>Guiding principles</strong>
        <ul>
          <li>
            When selecting the parent category is possible, it selects all the children below it.
          </li>
          <li>Behavior of the checkboxes is consistent with the checkbox component.</li>
          <li>Parent categories cannot have icons (at this point)</li>
          <li>
            In the event multiple items are selected, the top dropdown will display “multiple
            selected” (and NOT a list of all the selections made)
          </li>
        </ul>
      </Card>
    </ComponentSection>
  );
};

const CheckboxesSection: React.FC = () => {
  return (
    <ComponentSection id="Checkboxes" title="Checkboxes">
      <Card>
        <p>
          Checkboxes (<code>{'<Checkbox>'}</code>) give people a way to select one or more items from a group, or switch
          between two mutually exclusive options (checked or unchecked, on or off).
        </p>
      </Card>
      <Card title="Best practices">
        <strong>Layout</strong>
        <ul>
          <li>
            Use a single check box when there&apos;s only one selection to make or choice to
            confirm. Selecting a blank check box selects it. Selecting it again clears the check
            box.
          </li>
          <li>
            Use multiple check boxes when one or more options can be selected from a group. Unlike
            radio buttons, selecting one check box will not clear another check box.
          </li>
        </ul>
        <strong>Content</strong>
        <ul>
          <li>
            Separate two groups of check boxes with headings rather than positioning them one after
            the other.
          </li>
          <li>Use sentence-style capitalization—only capitalize the first word.</li>
          <li>
            Don&apos;t use end punctuation (unless the check box label absolutely requires multiple
            sentences).
          </li>
          <li>Use a sentence fragment for the label, rather than a full sentence.</li>
          <li>
            Make it easy for people to understand what will happen if they select or clear a check
            box.
          </li>
        </ul>
      </Card>
      <Card title="Usage">
        <strong>Basic checkboxes</strong>
        <Checkbox>This is a basic checkbox.</Checkbox>
        <strong>Guiding principles</strong>
        <ul>
          <li>8px right margin from the checkbox.</li>
          <li>5px vertical margins above and below the checkbox</li>
          <li>5px padding for mandatory and info icons</li>
          <li>One style of checkboxes throughout the experience.</li>
        </ul>
        <strong>Variations</strong>
        <Checkbox checked>Checked checkbox</Checkbox>
        <Checkbox checked={false}>Unchecked checkbox</Checkbox>
        <Checkbox checked disabled>
          Disabled checked checkbox
        </Checkbox>
        <p>Mandatory checkbox - not implemented.</p>
        <p>Mandatory checkbox with info sign - not implemented.</p>
        <Checkbox indeterminate>Indeterminate checkbox</Checkbox>
      </Card>
    </ComponentSection>
  );
};

const SearchboxesSection: React.FC = () => {
  return (
    <ComponentSection id="Searchboxes" title="Searchboxes">
      <Card>
        <p>
          A search box (<code>{'<Searchbox>'}</code>) provides an input field for searching content within a site or app
          to find specific items.
        </p>
      </Card>
      <Card title="Best practices">
        <strong>Layout</strong>
        <ul>
          <li>
            Don&apos;t build a custom search control based on the default text box or any other
            control.
          </li>
          <li>
            Use a search box without a parent container when it&apos;s not restricted to a certain
            width to accommodate other content. This search box will span the entire width of the
            space it&apos;s in.
          </li>
        </ul>
        <strong>Content</strong>
        <ul>
          <li>
            Use placeholder text in the search box to describe what people can search for. For
            example, &quot;Search&quot;, &quot;Search files&quot;, or &quot;Search contacts
            list&quot;.
          </li>
          <li>
            Although search entry points tend to be similarly visualized, they can provide access to
            results that range from broad to narrow. By effectively communicating the scope of a
            search, you can ensure that people&apos;s expectations are met by the capabilities of
            the search you&apos;re performing, which will reduce the possibility of frustration. The
            search entry point should be placed near the content being searched.
          </li>
        </ul>
      </Card>
      <Card title="Usage">
        <strong>Default Searchbox</strong>
        <Input.Search placeholder="input search text" style={{ width: 200 }} />
        <strong>Guiding principles</strong>
        <ul>
          <li>A user should always be able to cancel/clear out a search</li>
          <li>We need to provide feedback when a search is taking longer than expected</li>
          <li>Input box experience is from input box component</li>
        </ul>
        <strong>Variations</strong>
        <Input.Search allowClear enterButton style={{ width: 200 }} value="Active search box" />
        <Input.Search disabled placeholder="disabled search box" style={{ width: 200 }} />
        <hr />
        <strong>In-table Searchbox</strong>
        <p>Not implemented</p>
        <strong>Guiding principles</strong>
        <ul>
          <li>Search input box needs to be at least 30 characters long</li>
          <li>
            We need to provide feedback when a search is taking longer than expected (&gt;1.5 sec)
            or when its a long running operation
          </li>
        </ul>
        <hr />
        <strong>Search box with scopes</strong>
        <p>Not implemented</p>
        <strong>Guiding principles</strong>
        <ul>
          <li>Search input box needs to be at least 30 characters long</li>
          <li>
            We need to provide feedback when a search is taking longer than expected (&gt;1.5 sec)
            or when its a long running operation
          </li>
          <li>Dropdown component behavior is the same as the dropdown checkmark component</li>
        </ul>
      </Card>
    </ComponentSection>
  );
};

const SpinbuttonsSection: React.FC = () => {
  return (
    <ComponentSection id="Spinbuttons" title="Spinbuttons">
      <Card>
        <p>
          A spin button (<code>{'<Spinbutton>'}</code>) allows someone to incrementally adjust a value in small steps.
          It&apos;s mainly used for numeric values, but other values are supported too.
        </p>
      </Card>
      <Card title="Best practices">
        <strong>Layout</strong>
        <ul>
          <li>
            Place labels to the left of the spin button control. For example, &quot;Length of ruler
            (cm)&quot;.
          </li>
          <li>Spin button width should adjust to fit the number values.</li>
        </ul>
        <strong>Content</strong>
        <ul>
          <li>Use a spin button when you need to incrementally change a value.</li>
          <li>Use a spin button when values are tied to a unit of measure.</li>
          <li>Don&apos;t use a spin button for binary settings.</li>
          <li>Don&apos;t use a spin button for a range of three values or less.</li>
          <li>Include a label indicating what value the spin button changes.</li>
        </ul>
      </Card>
      <Card title="Usage">
        <strong>Default spin button</strong>
        <InputNumber defaultValue={3} />
        <strong>Guiding principles</strong>
        <ul>
          <li>It&apos;s only up-on hover that the up/down arrows become apparent.</li>
          <li>Number/type input starts 12px from border. </li>
          <li>When in focus, use arrow up/down of keyboard to facilitate spinning.</li>
        </ul>
        <hr />
        <strong>Variations and states of spin buttons</strong>
        <InputNumber disabled />
      </Card>
    </ComponentSection>
  );
};

const TextfieldsSection: React.FC = () => {
  return (
    <ComponentSection id="Textfields" title="Input - Textfields">
      <Card>
        <p>
          Text fields (<code>{'<Textfield>'}</code>) give people a way to enter and edit text. They&apos;re used in
          forms, modal dialogs, tables, and other surfaces where text input is required.
        </p>
      </Card>
      <Card title="Best practices">
        <strong>Layout</strong>
        <ul>
          <li>Use a multiline text field when long entries are expected.</li>
          <li>
            Don&apos;t place a text field in the middle of a sentence, because the sentence
            structure might not make sense in all languages. For example, &quot;Remind me in
            [textfield] weeks&quot; should instead read, &quot;Remind me in this many weeks:
            [textfield]&quot;.
          </li>
          <li>Format the text field for the expected entry.</li>
        </ul>
        <strong>Content</strong>
        <ul>
          <li>
            Include a short label above the text field to communicate what information should be
            entered. Don&apos;t use placeholder text instead of a label. Placeholder text poses a
            variety of accessibility issues (including possible problems with color/contrast, and
            people thinking the form input is already filled out).
          </li>
          <li>
            When part of a form, make it clear which fields are required vs. optional. If the input
            is required, add an asterisk &quot;*&quot; to the label. For screen readers, make sure
            the aria-required property is set to true.
          </li>
          <li>Use sentence-style capitalization—only capitalize the first word.</li>
          <li>
            ALL input fields need to have validation and limitations in-place (length of input
            especially)
          </li>
          <li>Field validation errors need to be provided for every mandatory field. </li>
        </ul>
      </Card>
      <Card title="Usage">
        <strong>Default input - text field</strong>
        <Input defaultValue="text" />
      </Card>
    </ComponentSection>
  );
};

const ListsSection: React.FC = () => {
  const mockColumns = [
    {
      dataIndex: 'id',
      sorter: true,
      title: 'ID',
    },
    {
      dataIndex: 'name',
      sorter: true,
      title: 'Name',
    },
  ];

  const mockRows = [
    {
      id: 'Row id',
      name: 'Row name',
    },
  ];

  return (
    <ComponentSection id="Lists" title="Lists (tables)">
      <Card>
        <p>
          A list (<code>{'<ResponsiveTable>'}</code>) is a robust way to display an information-rich collection of items, and
          allow people to sort, group, and filter the content. Use a details list when information
          density is critical.
        </p>
      </Card>
      <Card title="Best practices">
        <strong>Layout</strong>
        <ul>
          <li>
            List items are composed of selection, icon, and name columns at minimum. You can include
            other columns, such as date modified, or any other metadata field associated with the
            collection.
          </li>
          <li>
            Avoid using file type icon overlays to denote status of a file as it can make the entire
            icon unclear.
          </li>
          <li>
            If there are multiple lines of text in a column, consider the variable row height
            variant.
          </li>
          <li>Give columns ample default width to display information.</li>
        </ul>
        <strong>Content</strong>
        <ul>
          <li>
            Use sentence-style capitalization for column headers—only capitalize the first word.
          </li>
        </ul>
        <strong>Accessibility</strong>
        <ul>
          <li>
            When creating a DetailsList where one column is clearly the primary label for the row,
            it&apos;s best to use isRowHeader on that column to create a better screen reader
            experience navigating the table. For selectable DetailsLists, specifying a row header
            also gives the checkboxes a better accessible label.
          </li>
        </ul>
        <strong>Keyboard hotkeys</strong>
        <ul>
          <li>
            DetailsList supports different selection modes with keyboard behavior differing based on
            the current selection mode.
          </li>
        </ul>
      </Card>
      <Card title="Usage">
        <strong>Default list</strong>
        <ResponsiveTable columns={mockColumns} dataSource={mockRows} rowKey="id" />
      </Card>
    </ComponentSection>
  );
};

const BreadcrumbsSection: React.FC = () => {
  return (
    <ComponentSection id="Breadcrumbs" title="Breadcrumbs">
      <Card>
        <p>
          <code>{'<Breadcrumb>'}</code>s should be used as a navigational aid in your app or site. They indicate the
          current page&apos;s location within a hierarchy and help the user understand where they
          are in relation to the rest of that hierarchy. They also afford one-click access to higher
          levels of that hierarchy.
        </p>
        <p>
          Breadcrumbs are typically placed, in horizontal form, under the masthead or navigation of
          an experience, above the primary content area.
        </p>
      </Card>
      <Card title="Best practices">
        <strong>Accessibility</strong>
        <ul>
          <li>By default, Breadcrumb uses arrow keys to cycle through each item. </li>
          <li>
            Place Breadcrumbs at the top of a page, above a list of items, or above the main content
            of a page.{' '}
          </li>
        </ul>
      </Card>
      <Card title="Usage">
        <strong>Breadcrumb</strong>
        <Breadcrumb>
          <Breadcrumb.Item>Level 0</Breadcrumb.Item>
          <Breadcrumb.Item>Level 1</Breadcrumb.Item>
          <Breadcrumb.Item>Level 2</Breadcrumb.Item>
        </Breadcrumb>
      </Card>
    </ComponentSection>
  );
};

const FacepileSection: React.FC = () => {
  return (
    <ComponentSection id="Facepile" title="Facepile">
      <Card>
        <p>
          A face pile (<code>{'<Facepile>'}</code>) displays a list of personas. Each circle represents a person and
          contains their image or initials. Often this control is used when sharing who has access to
          a specific view or file, or when assigning someone a task within a workflow.
        </p>
      </Card>
      <Card title="Best practices">
        <strong>Content considerations</strong>
        <ul>
          <li>
            The face pile empty state should only include an &quot;Add&quot; button. Another variant
            is to use an input field with placeholder text instructing people to add a person. See
            the people picker component for the menu used to add people to the face pile list.
          </li>
          <li>
            When there is only one person in the face pile, consider using their name next to the
            face or initials.
          </li>
          <li>
            When there is a need to show the face pile expanded into a vertical list, include a
            downward chevron button. Selecting the chevron opens a standard list view of people.
          </li>
          <li>
            When the face pile exceeds a max number of 5 people, show a button at the end of the
            list indicating how many are not being shown. Clicking or tapping on the overflow would
            open a standard list view of people.
          </li>
          <li>
            The component can include an &quot;Add&quot; button which can be used for quickly adding
            a person to the list.
          </li>
          <li>
            When hovering over a person in the face pile, include a tooltip or people card that
            offers more information about that person.
          </li>
        </ul>
      </Card>
      <Card title="Usage">
        <strong>Facepile</strong>
        <UserAvatar />
        <strong>Guiding principles</strong>
        <ul>
          <li>Each facepile item needs to have its own color</li>
          <li>4px of space between faces</li>
          <li>If more than 4 faces: add a “+ n” where the 5th face would be.</li>
        </ul>
        <strong>Variations</strong>
        <ul>
          <li>Facepile with 8 people</li>
          <p>Not implemented</p>
          <li>Facepile with both name initials</li>
          <p>Not implemented</p>
        </ul>
      </Card>
    </ComponentSection>
  );
};

const ActionBarSection: React.FC = () => {
  const { experiment } = generateTestExperimentData();
  return (
    <ComponentSection id="ActionBar" title="ActionBar">
      <Card>
        <p>
          <code>{'<ActionBar>'}</code> is a surface that houses commands that operate on the content of the window,
          panel, or parent region it resides above. ActionBar are one of the most visible and
          recognizable ways to surface commands, and can be an intuitive method for interacting with
          content on the page; however, if overloaded or poorly organized, they can be difficult to
          use and hide valuable commands from your user. ActionBar can also display a search box for
          finding content, hold simple commands as well as menus, or display the status of ongoing
          actions.
        </p>
        <p>
          Commands should be sorted in order of importance, from left-to-right or right-to-left
          depending on the culture. Secondarily, organize commands in logical groupings for easier
          recall. ActionBars work best when they display no more than 5-7 commands. This helps users
          quickly find your most valuable features. If you need to show more commands, consider
          using the overflow menu. If you need to render status or viewing controls, these go on the
          right side of the ActionBar (or left side if in a left-to-right experience). Do not
          display more than 2-3 items on the right side as it will make the overall ActionBar
          difficult to parse.
        </p>
        <p>
          All command items should have an icon and a label. Commands can render as labels only as
          well. In smaller widths, commands can just use icon only, but only for the most
          recognizable and frequently used commands. All other commands should go into an overflow
          where text labels can be shown.
        </p>
      </Card>
      <Card title="Best practices">
        <strong>Content considerations</strong>
        <ul>
          <li>
            Sort commands in order of importance from left to right or right to left depending on
            the culture.
          </li>
          <li>Use overflow to house less frequently-used commands.</li>
          <li>
            In small breakpoints, only have the most recognizable commands render as icon only.
          </li>
        </ul>
      </Card>
      <Card title="Usage">
        <strong>Actionbar defaults</strong>
        <ExperimentDetailsHeader
          experiment={experiment}
          fetchExperimentDetails={() => {
            return;
          }}
        />
      </Card>
    </ComponentSection>
  );
};

const PivotSection: React.FC = () => {
  return (
    <ComponentSection id="Pivot" title="Pivot">
      <Card>
        <p>
          The Pivot control (<code>{'<Tabs>'}</code>) and related tabs pattern are used for navigating frequently accessed,
          distinct content categories. Pivots allow for navigation between two or more content views
          and relies on text headers to articulate the different sections of content.
        </p>
        <p>Tapping on a pivot item header navigates to that header&apos;s section content.</p>
        <p>
          Tabs are a visual variant of Pivot that use a combination of icons and text or just icons
          to articulate section content.
        </p>
      </Card>
      <Card title="Best practices">
        <strong>Content considerations</strong>
        <ul>
          <li>
            Be concise on the navigation labels, ideally one or two words rather than a phrase.
          </li>
          <li>
            Use on content-heavy pages that require a significant amount of scrolling to access the
            various sections.
          </li>
        </ul>
      </Card>
      <Card title="Usage">
        <strong>Pivot</strong>
        <Pivot
          items={[
            { key: 'Overview', label: 'Overview' },
            { key: 'hyperparameters', label: 'Hyperparameters' },
            { key: 'checkpoints', label: 'Checkpoints' },
            { key: 'code', label: 'Code' },
            { key: 'notes', label: 'Notes' },
            { key: 'profiler', label: 'Profiler' },
            { key: 'logs', label: 'Logs' },
          ]}
        />
      </Card>
    </ComponentSection>
  );
};

const PaginationSection: React.FC = () => {
  return (
    <ComponentSection id="Pagination" title="Pagination">
      <Card>
        <p>
          <code>{'<Pagination>'}</code> is the process of splitting the contents of a website, or section of contents
          from a website, into discrete pages. This user interface design pattern is used so users are
          not overwhelmed by a mass of data on one page. Page breaks are automatically set.
        </p>
      </Card>
      <Card title="Best practices">
        <strong>Content considerations</strong>
        <ul>
          <li>Use ordinal numerals or letters of the alphabet.</li>
          <li>
            Indentify the current page in addition to the pages in immediate context/surrounding.
          </li>
        </ul>
      </Card>
      <Card title="Usage">
        <strong>Pagination default</strong>
        <Pagination total={500} />
        <strong>Considerations</strong>
        <ul>
          <li>
            Always give the user the option to navigate to the first & last page -- this helps with
            sorts.
          </li>
          <li>
            Provide the user with 2 pages before/after when navigating &apos;island&apos;
            page-counts.
          </li>
          <li>Provide the user with the first 4 or last 4 pages of the page-range.</li>
          <li>
            Ensure the FocusTrap is set to the whole pagination component so that user doesn&apos;t
            tabs in/out accidentally.
          </li>
        </ul>
      </Card>
    </ComponentSection>
  );
};

const DataCardsSection: React.FC = () => {
  const rps = resourcePools as unknown as ResourcePool[];

  return (
    <ComponentSection id="DataCards" title="DataCards">
      <Card>
        <p>
          A DataCard (<code>{'<OverviewStats>'}</code>) contains additional metadata or actions. This offers people a richer view into a
          file than the typical grid view.
        </p>
      </Card>
      <Card title="Best practices">
        <strong>Content considerations</strong>
        <ul>
          <li>Incorporate metadata that is relevant and useful in this particular view.</li>
          <li>
            Don&apos;t use a document card in views where someone is likely to be performing bulk
            operations in files, or when the list may get very long. Specifically, if you&apos;re
            showing all the items inside an actual folder, a card may be overkill because the
            majority of the items in the folder may not have interesting metadata.
          </li>
          <li>
            Don&apos;t use a document card if space is at a premium or you can&apos;t show relevant
            and timely commands or metadata. Cards are useful because they can expose on-item
            interactions like “Share” buttons or view counts.
          </li>
        </ul>
      </Card>
      <Card title="Usage">
        <strong>DataCard default</strong>
        <Grid gap={ShirtSize.Medium} minItemWidth={180} mode={GridMode.AutoFill}>
          <OverviewStats title="Last Runner State">Validating</OverviewStats>
          <OverviewStats title="Start time">7 mo ago</OverviewStats>
          <OverviewStats title="Total Checkpoint size">14.4 MB</OverviewStats>
          <OverviewStats clickable title="Best Checkpoint">
            Batch 1000
          </OverviewStats>
        </Grid>
        <strong>Considerations</strong>
        <ul>
          <li>Ensure links are tab-able.</li>
          <li>Ensure data is relevant and if not, remove it.</li>
          <li>We need to revisit the density of each of the cards and content.</li>
          <li>
            Implement quick actions inside of the card as to prevent the user from providing
            additional clicks.
          </li>
        </ul>
        <strong>DataCard variations</strong>
        <ul>
          <li>Resource pool card (<code>{'<ResourcePoolCard>'}</code>)</li>
          <ResourcePoolCard resourcePool={rps[0]} />
        </ul>
      </Card>
    </ComponentSection>
  );
};

const LogViewerSection: React.FC = () => {
  const sampleLogs = [
    {
      id: 1,
      level: V1LogLevel.INFO,
      message: 'Determined master 0.19.7-dev0 (built with go1.18.7)',
      timestamp: '2022-06-02T21:48:07.456381-06:00',
    },
    {
      id: 2,
      level: V1LogLevel.INFO,
      message:
        'connecting to database determined-master-database-tytmqsutj5d1.cluster-csrkoc1nkoog.us-west-2.rds.amazonaws.com:5432',
      timestamp: '2022-07-02T21:48:07.456381-06:00',
    },
    {
      id: 3,
      level: V1LogLevel.INFO,
      message:
        'running DB migrations from file:///usr/share/determined/master/static/migrations; this might take a while...',
      timestamp: '2022-08-02T21:48:07.456381-06:00',
    },
    {
      id: 4,
      level: V1LogLevel.INFO,
      message: 'no migrations to apply; version: 20221026124235',
      timestamp: '2022-09-02T21:48:07.456381-06:00',
    },
    {
      id: 5,
      level: V1LogLevel.ERROR,
      message:
        'failed to aggregate resource allocation: failed to add aggregate allocation: ERROR: range lower bound must be less than or equal to range upper bound (SQLSTATE 22000)  actor-local-addr="allocation-aggregator" actor-system="master" go-type="allocationAggregator"',
      timestamp: '2022-10-02T21:48:07.456381-06:00',
    },
    {
      id: 6,
      level: V1LogLevel.WARNING,
      message:
        'received update on unknown agent  actor-local-addr="aux-pool" actor-system="master" agent-id="i-018fadb36ddbfe97a" go-type="ResourcePool" resource-pool="aux-pool"',
      timestamp: '2022-11-02T21:48:07.456381-06:00',
    },
  ];
  return (
    <ComponentSection id="LogViewer" title="LogViewer">
      <Card>
        <p>
          A Logview (<code>{'<LogViewer>'}</code>) prints events that have been configured to be triggered and return them to the
          user in a running stream.
        </p>
      </Card>
      <Card title="Best practices">
        <strong>Content considerations</strong>
        <ul>
          <li>
            Prioritize accessibility and readability of the log entry as details can always be
            generated afterwards.
          </li>
          <li>
            Prioritize IntelliSense type of readability improvements as it helps scannability of the
            text.
          </li>
          <li>Provide the user with ways of searching & filtering down logs.</li>
        </ul>
      </Card>
      <Card title="Usage">
        <strong>LogViewer default</strong>
        <div style={{ height: '300px' }}>
          <LogViewer decoder={mapV1LogsResponse} initialLogs={sampleLogs} sortKey="id" />
        </div>
        <strong>Considerations</strong>
        <ul>
          <li>
            Ensure that we don&apos;t overload the users with information --&gt; we need to know
            what events we&apos;re listening to.
          </li>
          <li>Ensure the capability of searching/filtering log entries.</li>
        </ul>
      </Card>
    </ComponentSection>
  );
};

const TooltipsSection: React.FC = () => {
  const text = 'Tooltip text';
  const buttonWidth = 70;

  return (
    <ComponentSection id="Tooltips" title="Tooltips">
      <Card>
        <p>
          A good tooltip (<code>{'<Tooltip>'}</code>) briefly describes unlabeled controls or provides a bit of additional
          information about labeled controls, when this is useful. It can also help customers navigate
          the UI by offering additional—not redundant—information about control labels, icons, and
          links. A tooltip should always add valuable information; use sparingly.
        </p>
      </Card>
      <Card title="Best practices">
        <strong>Content</strong>
        <ul>
          <li>
            Don&apos;t use a tooltip to restate a button name that&apos;s already shown in the UI.
          </li>
          <li>
            When a control or UI element is unlabeled, use a simple, descriptive noun phrase. For
            Only use periods for complete sentences.italize the first word (unless a subsequent word
            is a proper noun), and don&apos;t use a period.
          </li>
          <li>
            For a disabled control that could use an explanation, provide a brief description of the
            state in which the control will be enabled. For example: “This feature is available for
            line charts.”
          </li>
          <li>Only use periods for complete sentences.</li>
        </ul>
      </Card>
      <Card title="Usage">
        <strong>Tooltips default</strong>
        <Space>
          <Tooltip title={text}>
            <span>Trigger on hover</span>
          </Tooltip>
          <Tooltip title={text} trigger="click">
            <span>Trigger on click</span>
          </Tooltip>
          <Tooltip title={text} trigger="contextMenu">
            <span>Trigger on right click</span>
          </Tooltip>
        </Space>
        <strong>Considerations</strong>
        <ul>
          <li>
            Nest the tooltip where the content in a cell/text is. Don’t let it levitate in the
            nothingness.
          </li>
        </ul>
        <strong>Variations</strong>
        <div>
          <div style={{ marginLeft: buttonWidth, whiteSpace: 'nowrap' }}>
            <Tooltip placement="topLeft" title={text}>
              <Button>TL</Button>
            </Tooltip>
            <Tooltip placement="top" title={text}>
              <Button>Top</Button>
            </Tooltip>
            <Tooltip placement="topRight" title={text}>
              <Button>TR</Button>
            </Tooltip>
          </div>
          <div style={{ float: 'left', width: buttonWidth }}>
            <Tooltip placement="leftTop" title={text}>
              <Button>LT</Button>
            </Tooltip>
            <Tooltip placement="left" title={text}>
              <Button>Left</Button>
            </Tooltip>
            <Tooltip placement="leftBottom" title={text}>
              <Button>LB</Button>
            </Tooltip>
          </div>
          <div style={{ marginLeft: buttonWidth * 4 + 24, width: buttonWidth }}>
            <Tooltip placement="rightTop" title={text}>
              <Button>RT</Button>
            </Tooltip>
            <Tooltip placement="right" title={text}>
              <Button>Right</Button>
            </Tooltip>
            <Tooltip placement="rightBottom" title={text}>
              <Button>RB</Button>
            </Tooltip>
          </div>
          <div style={{ clear: 'both', marginLeft: buttonWidth, whiteSpace: 'nowrap' }}>
            <Tooltip placement="bottomLeft" title={text}>
              <Button>BL</Button>
            </Tooltip>
            <Tooltip placement="bottom" title={text}>
              <Button>Bottom</Button>
            </Tooltip>
            <Tooltip placement="bottomRight" title={text}>
              <Button>BR</Button>
            </Tooltip>
          </div>
        </div>
      </Card>
    </ComponentSection>
  );
};

const DesignKit: React.FC = () => {
  const { actions } = useUI();

  useEffect(() => {
    actions.hideChrome();
  }, [actions]);

  return (
    <Page bodyNoPadding docTitle="Design Kit">
      <div className={css.base}>
        <nav>
          <Link reloadDocument to={{}}>
            <Logo branding={BrandingType.Determined} orientation="horizontal" />
          </Link>
          <ThemeToggle />
          <ul>
            {componentOrder.map((componentId) => (
              <li key={componentId}>
                <Link reloadDocument to={`#${componentId}`}>
                  {Components[componentId]}
                </Link>
              </li>
            ))}
          </ul>
        </nav>
        <main>
          <ButtonsSection />
          <DropdownsSection />
          <CheckboxesSection />
          <SearchboxesSection />
          <SpinbuttonsSection />
          <TextfieldsSection />
          <ListsSection />
          <BreadcrumbsSection />
          <FacepileSection />
          <ActionBarSection />
          <PivotSection />
          <PaginationSection />
          <DataCardsSection />
          <LogViewerSection />
          <TooltipsSection />
        </main>
      </div>
    </Page>
  );
};

export default DesignKit;
