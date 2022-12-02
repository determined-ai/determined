import { PoweroffOutlined } from '@ant-design/icons';
import { Alert, Button, Card, Checkbox, Input, InputNumber, Space } from 'antd';
import React, { useEffect } from 'react';
import { Link } from 'react-router-dom';

import Label from 'components/Label';
import Logo from 'components/Logo';
import SelectFilter from 'components/SelectFilter';
import ThemeToggle from 'components/ThemeToggle';
import Icon from 'shared/components/Icon';
import useUI from 'shared/contexts/stores/UI';
import { ValueOf } from 'shared/types';
import { BrandingType } from 'types';

import css from './DesignKit.module.scss';

const Components = {
  Buttons: 'Buttons',
  Checkboxes: 'Checkboxes',
  Dropdowns: 'Comboboxes & Dropdowns',
  Labels: 'Labels',
  Searchboxes: 'Search boxes',
  Spinbuttons: 'Spin buttons',
  Textfields: 'Input - text fields',
} as const;

type ComponentNames = ValueOf<typeof Components>;
type ComponentIds = keyof typeof Components;

const componentOrder: ComponentIds[] = [
  'Buttons',
  'Dropdowns',
  'Checkboxes',
  'Labels',
  'Searchboxes',
  'Spinbuttons',
  'Textfields',
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

const ReviewAlert: React.FC = () => {
  return (
    <Alert
      message="Caution: this component is currently under review. Expect this component to undergo
    updates in near-term releases."
      showIcon
      type="warning"
    />
  );
};

const ButtonsSection: React.FC = () => {
  return (
    <ComponentSection id="Buttons" title="Buttons">
      <ReviewAlert />
      <Card>
        Buttons give people a way to trigger an action. They&apos;re typically found in forms,
        dialog panels, and dialogs. Some buttons are specialized for particular tasks, such as
        navigation, repeated actions, or presenting menus.
      </Card>
      <Card title="Design audit">
        <strong>
          This component is currently under review and will receive updates to address:
        </strong>
        <ul>
          <li>Font inconsistency</li>
          <li>Internal padding inconsistencies</li>
          <li>Button states do not meet accessibility requirements.</li>
        </ul>
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
      <ReviewAlert />
      <Card>
        A dropdown/combo box combines a text field and a dropdown giving people a way to select an
        option from a list or enter their own choice.
      </Card>
      <Card title="Design audit">
        <strong>
          This component is currently under review and will receive updates to address:
        </strong>
        <ul>
          <li>Font inconsistency</li>
          <li>Internal padding inconsistencies</li>
          <li>Button states do not meet accessibility requirements.</li>
        </ul>
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
      <ReviewAlert />
      <Card>
        Check boxes (Checkbox) give people a way to select one or more items from a group, or
        switch between two mutually exclusive options (checked or unchecked, on or off).
      </Card>
      <Card title="Design audit">
        <strong>
          This component is currently under review and will receive updates to address:
        </strong>
        <ul>
          <li>Font inconsistency</li>
          <li>Internal padding inconsistencies</li>
          <li>Button states do not meet accessibility requirements.</li>
        </ul>
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
            Use multiple check boxes when one or more options can be selected from a group.
            Unlike radio buttons, selecting one check box will not clear another check box.
          </li>
        </ul>
        <strong>Content</strong>
        <ul>
          <li>
            Separate two groups of check boxes with headings rather than positioning them one
            after the other.
          </li>
          <li>Use sentence-style capitalization—only capitalize the first word.</li>
          <li>
            Don&apos;t use end punctuation (unless the check box label absolutely requires
            multiple sentences).
          </li>
          <li>Use a sentence fragment for the label, rather than a full sentence.</li>
          <li>
            Make it easy for people to understand what will happen if they select or clear a
            check box.
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

const LabelsSection: React.FC = () => {
  return (
    <ComponentSection id="Labels" title="Labels">
      <ReviewAlert />
      <Card>
        Labels give a name or title to a control or group of controls, including text fields,
        check boxes, combo boxes, radio buttons, and drop-down menus.
      </Card>
      <Card title="Design audit">
        <strong>
          This component is currently under review and will receive updates to address:
        </strong>
        <ul>
          <li>Font inconsistency</li>
          <li>Internal padding inconsistencies</li>
          <li>Button states do not meet accessibility requirements.</li>
        </ul>
      </Card>
      <Card title="Best practices">
        <strong>Layout</strong>
        <ul>
          <li>Labels should be close to the control they&apos;re paired with.</li>
        </ul>
        <strong>Content</strong>
        <ul>
          <li>Labels should describe the purpose of the control.</li>
          <li>Use sentence-style capitalization—only capitalize the first word.</li>
          <li>Be short and concise.</li>
          <li>Use nouns or short noun phrases.</li>
          <li>
            Don&apos;t use labels as instructional text. For example, &quot;Click to get
            started&quot;.
          </li>
        </ul>
      </Card>
      <Card title="Usage">
        <strong>Basic labels</strong>
        <Label>I am a label.</Label>
        <strong>Guiding principles</strong>
        <ul>
          <li>8px right margin from the checkbox.</li>
          <li>5px vertical margins above and below the checkbox</li>
          <li>5px padding for mandatory and info icons</li>
          <li>One style of checkboxes throughout the experience.</li>
        </ul>
        <strong>Variations</strong>
        <p>Bold mandatory label - not implemented</p>
        <p>Disabled label - not implemented</p>
        <p>Mandatory label with an explanation - not implemented</p>
      </Card>
    </ComponentSection>
  );
};

const SearchboxesSection: React.FC = () => {
  return (
    <ComponentSection id="Searchboxes" title="Search boxes">
      <ReviewAlert />
      <Card>
        A search box (SearchBox) provides an input field for searching content within a site or
        app to find specific items.
      </Card>
      <Card title="Design audit">
        <strong>
          This component is currently under review and will receive updates to address:
        </strong>
        <ul>
          <li>Font inconsistency</li>
          <li>Internal padding inconsistencies</li>
          <li>Button states do not meet accessibility requirements.</li>
        </ul>
      </Card>
      <Card title="Best practices">
        <strong>Layout</strong>
        <ul>
          <li>
            Don&apos;t build a custom search control based on the default text box or any other
            control.
          </li>
          <li>
            Use a search box without a parent container when it&apos;s not restricted to a
            certain width to accommodate other content. This search box will span the entire
            width of the space it&apos;s in.
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
            Although search entry points tend to be similarly visualized, they can provide
            access to results that range from broad to narrow. By effectively communicating the
            scope of a search, you can ensure that people&apos;s expectations are met by the
            capabilities of the search you&apos;re performing, which will reduce the possibility
            of frustration. The search entry point should be placed near the content being
            searched.
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
            We need to provide feedback when a search is taking longer than expected (&gt;1.5
            sec) or when its a long running operation
          </li>
        </ul>
        <hr />
        <strong>Search box with scopes</strong>
        <p>Not implemented</p>
        <strong>Guiding principles</strong>
        <ul>
          <li>Search input box needs to be at least 30 characters long</li>
          <li>
            We need to provide feedback when a search is taking longer than expected (&gt;1.5
            sec) or when its a long running operation
          </li>
          <li>Dropdown component behavior is the same as the dropdown checkmark component</li>
        </ul>
      </Card>
    </ComponentSection>
  );
};

const SpinbuttonsSection: React.FC = () => {
  return (
    <ComponentSection id="Spinbuttons" title="Spin buttons">
      <ReviewAlert />
      <Card>
        A spin button (SpinButton) allows someone to incrementally adjust a value in small
        steps. It&apos;s mainly used for numeric values, but other values are supported too.
      </Card>
      <Card title="Design audit">
        <strong>
          This component is currently under review and will receive updates to address:
        </strong>
        <ul>
          <li>
            Overall value-add to the experience --&gt; removing this component requires less
            maintenance.{' '}
          </li>
        </ul>
      </Card>
      <Card title="Best practices">
        <strong>Layout</strong>
        <ul>
          <li>
            Place labels to the left of the spin button control. For example, &quot;Length of
            ruler (cm)&quot;.
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
    <ComponentSection id="Textfields" title="Input - text fields">
      <ReviewAlert />
      <Card>
        Text fields (TextField) give people a way to enter and edit text. They&apos;re used in
        forms, modal dialogs, tables, and other surfaces where text input is required.
      </Card>
      <Card title="Design audit">
        <strong>
          This component is currently under review and will receive updates to address:
        </strong>
        <ul>
          <li>Component seems to not be used consistently.</li>
        </ul>
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
            entered. Don&apos;t use placeholder text instead of a label. Placeholder text poses
            a variety of accessibility issues (including possible problems with color/contrast,
            and people thinking the form input is already filled out).
          </li>
          <li>
            When part of a form, make it clear which fields are required vs. optional. If the
            input is required, add an asterisk &quot;*&quot; to the label. For screen readers,
            make sure the aria-required property is set to true.
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

const DesignKit: React.FC = () => {
  const { actions } = useUI();

  useEffect(() => {
    actions.hideChrome();
  }, [actions]);

  return (
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
        <LabelsSection />
        <SearchboxesSection />
        <SpinbuttonsSection />
        <TextfieldsSection />
      </main>
    </div>
  );
};

export default DesignKit;
