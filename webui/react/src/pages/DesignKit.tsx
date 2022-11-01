import { PoweroffOutlined } from '@ant-design/icons';
import { Alert, Button, Card, Checkbox, Space } from 'antd';
import React, { useEffect } from 'react';
import { Link } from 'react-router-dom';

import Logo from 'components/Logo';
import SelectFilter from 'components/SelectFilter';
import ThemeToggle from 'components/ThemeToggle';
import Icon from 'shared/components/Icon';
import useUI from 'shared/contexts/stores/UI';
import { BrandingType } from 'types';

import css from './DesignKit.module.scss';

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
          <li>
            <Link reloadDocument to="#buttons_anchor">
              Buttons
            </Link>
          </li>
          <li>
            <Link reloadDocument to="#dropdowns_anchor">
              Dropdowns
            </Link>
          </li>
          <li>
            <Link reloadDocument to="#checkboxes_anchor">
              Checkboxes
            </Link>
          </li>
        </ul>
      </nav>
      <main>
        <section>
          <h3 id="buttons_anchor">Buttons</h3>
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
              <li>
                For single-page forms and focused tasks, left-align buttons with the container.
              </li>
              <li>
                Always place the primary button on the left, the secondary button just to the right
                of it.
              </li>
              <li>
                Show only one primary button that inherits theme color at rest state. If there are
                more than two buttons with equal priority, all buttons should have neutral
                backgrounds.
              </li>
              <li>
                Don&apos;t use a button to navigate to another place; use a link instead. The
                exception is in a wizard where &quot;Back&quot; and &quot;Next&quot; buttons may be
                used.
              </li>
              <li>
                Don&apos;t place the default focus on a button that destroys data. Instead, place
                the default focus on the button that performs the &quot;safe act&quot; and retains
                the content (such as &quot;Save&quot;) or cancels the action (such as
                &quot;Cancel&quot;).
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
        </section>
        <section>
          <h3 id="dropdowns_anchor">Comboboxes & Dropdowns</h3>
          <ReviewAlert />
          <Card>
            A dropdown/combo box combines a text field and a dropdown giving people a way to select
            an option from a list or enter their own choice.
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
                Use a combo box when there are multiple choices that can be collapsed under one
                title, when the list of items is long, or when space is constrained.
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
                ComboBox dropdowns render in their own layer by default to ensure they are not
                clipped by containers with overflow: hidden or overflow: scroll. This causes extra
                difficulty for people who use screen readers, so we recommend rendering the ComboBox
                options dropdown inline unless they are in overflow containers.
              </li>
            </ul>
            <strong>Truncation</strong>
            <ul>
              <li>
                By default, the ComboBox truncates option text instead of wrapping to a new line.
                Because this can lose meaningful information, it is recommended to adjust styles to
                wrap the option text.
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
                The parent cell has visually distinctive features to make it stand out in a
                multi-parent list from children.
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
                When selecting the parent category is possible, it selects all the children below
                it.
              </li>
              <li>Behavior of the checkboxes is consistent with the checkbox component.</li>
              <li>Parent categories cannot have icons (at this point)</li>
              <li>
                In the event multiple items are selected, the top dropdown will display “multiple
                selected” (and NOT a list of all the selections made)
              </li>
            </ul>
          </Card>
        </section>
        <section>
          <h3 id="checkboxes_anchor">Checkboxes</h3>
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
        </section>
      </main>
    </div>
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

export default DesignKit;
