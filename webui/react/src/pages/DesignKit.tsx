import { PoweroffOutlined } from '@ant-design/icons';
import { Button, Card, Space } from 'antd';
import React, { useEffect } from 'react';
import { Link } from 'react-router-dom';

import Logo from 'components/Logo';
import ThemeToggle from 'components/ThemeToggle';
import Icon from 'shared/components/Icon';
import useUI from 'shared/contexts/stores/UI';
import { BrandingType } from 'types';

const DesignKit: React.FC = () => {
  const { actions } = useUI();

  useEffect(() => {
    actions.hideChrome();
  }, [actions]);

  return (
    <div style={{ display: 'flex', gap: 8, height: '100%', overflow: 'auto' }}>
      <nav
        style={{
          backgroundColor: 'var(--theme-surface)',
          boxShadow: 'var(--theme-elevation)',
          display: 'flex',
          flexDirection: 'column',
          gap: 8,
          height: '100%',
          padding: 16,
          position: 'sticky',
          top: 0,
          width: 'var(--nav-side-bar-width-max)',
        }}>
        <Link reloadDocument to={{}}>
          <Logo branding={BrandingType.Determined} orientation="horizontal" />
        </Link>
        <ThemeToggle />
        <Link reloadDocument to="#buttons_anchor">
          Buttons
        </Link>
      </nav>
      <main style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        <h3 id="buttons_anchor">Buttons</h3>
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
        <Card bodyStyle={{ display: 'flex', flexDirection: 'column', gap: 8 }} title="Usage">
          <strong>Default Button</strong>
          <Space>
            <Button type="primary">Primary</Button>
            <Button>Secondary</Button>
          </Space>
          <strong>Guiding principles</strong>
          <ul>
            <li>15px inner horizontal padding</li>
            <li>8px inner vertical padding</li>
            <li>8px external margins</li>
            <li style={{ color: 'var(--theme-status-critical)' }}>
              Colors do not meet accessibility guidelines
            </li>
          </ul>
          <hr style={{ outline: 'solid var(--theme-stage-border-weak) 1px', width: '100%' }} />
          <strong>Default Button with icon</strong>
          <Space>
            <Button icon={<PoweroffOutlined />} type="primary">
              ButtonWithIcon
            </Button>
            <Button icon={<PoweroffOutlined />}>ButtonWithIcon</Button>
          </Space>
          <strong>Guiding principles</strong>
          <ul>
            <li>15px inner horizontal padding</li>
            <li>8px inner vertical padding</li>
            <li>8px padding between icon and text</li>
            <li>8px external margins</li>
            <li style={{ color: 'var(--theme-status-critical)' }}>
              Colors do not meet accessibility guidelines
            </li>
          </ul>
          <hr style={{ outline: 'solid var(--theme-stage-border-weak) 1px', width: '100%' }} />
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
          </Space>
          <strong>Guiding principles</strong>
          <ul>
            <li>Component needs to be reviewed/looked at.</li>
            <li>Missing distinguishing states</li>
            <li>Visual density</li>
          </ul>
        </Card>
      </main>
    </div>
  );
};

export default DesignKit;
