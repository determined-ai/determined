import { Button, Dropdown, Menu, Tooltip } from 'antd';
import React, { MouseEventHandler, useCallback, useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';

import Auth from 'contexts/Auth';
import UI from 'contexts/UI';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import { setupUrlForDev } from 'routes';
import { createNotebook } from 'services/api';
import { handlePath, openBlank } from 'utils/routes';
import { commandToTask } from 'utils/types';

import Avatar from './Avatar';
import Icon from './Icon';
import Link from './Link';
import css from './Navigation.module.scss';

interface ItemProps {
  badge?: number;
  icon: string;
  label: string;
  path?: string;
  popout?: boolean;
  onClick?: MouseEventHandler;
}

const NavigationItem: React.FC<ItemProps> = (props: ItemProps) => {
  const ui = UI.useStateContext();
  const location = useLocation();
  const [ isActive, setIsActive ] = useState(false);
  const classes = [ css.navItem ];

  if (ui.collapseChrome) classes.push(css.collapsed);
  if (isActive) classes.push(css.active);

  useEffect(() => {
    setIsActive(location.pathname === props.path);
  }, [ classes, location.pathname, props.path ]);

  const handleClick = useCallback((event: React.MouseEvent) => {
    handlePath(event, { onClick: props.onClick, path: props.path, popout: props.popout });
  }, [ props.onClick, props.path, props.popout ]);

  return <div className={classes.join(' ')} onClick={handleClick}>
    <Icon name={props.icon} size="large" />
    <div className={css.label}>{props.label}</div>
  </div>;
};

const Navigation: React.FC = () => {
  const { isAuthenticated, user } = Auth.useStateContext();
  const ui = UI.useStateContext();
  const setUI = UI.useActionContext();
  const [ isShowingCpu, setIsShowingCpu ] = useState(false);
  const classes = [ css.base ];

  const showNavigation = isAuthenticated && ui.showChrome;
  const version = process.env.VERSION;
  const isVersionLong = (version?.match(/\./g) || []).length > 2;
  const username = user?.username || 'Anonymous';

  if (ui.collapseChrome) classes.push(css.collapsed);

  const launchNotebook = useCallback(async (slots: number) => {
    try {
      const notebook = await createNotebook({ slots });
      const task = commandToTask(notebook);
      if (task.url) openBlank(setupUrlForDev(task.url));
      else throw new Error('Notebook URL not available.');
    } catch (e) {
      handleError({
        error: e,
        level: ErrorLevel.Error,
        message: e.message,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to Launch Notebook',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, []);

  const handleNotebookLaunch = useCallback(() => launchNotebook(1), [ launchNotebook ]);
  const handleCpuNotebookLaunch = useCallback(() => launchNotebook(0), [ launchNotebook ]);
  const handleVisibleChange = useCallback((visible: boolean) => setIsShowingCpu(visible), []);

  const handleCollapse = useCallback(() => {
    setUI({ type: UI.ActionType.ToggleChromeCollapse });
  }, [ setUI ]);

  return showNavigation ? (
    <nav className={classes.join(' ')}>
      <header>
        <div className={css.logo}>
          <div className={css.logoIcon} />
          <div className={css.logoLabel} />
        </div>
        <div className={css.version}>
          {isVersionLong ? (
            <Tooltip placement="right" title={`Version ${version}`}>
              <span className={css.versionLabel}>{version}</span>
            </Tooltip>
          ) : (
            <span className={css.versionLabel}>{version}</span>
          )}
        </div>
      </header>
      <main>
        <section className={css.launch}>
          <div className={css.launchBlock}>
            <Button
              className={css.launchButton}
              onClick={handleNotebookLaunch}>Launch Notebook</Button>
            <Dropdown
              arrow
              overlay={(
                <Menu>
                  {ui.collapseChrome &&
                    <Menu.Item onClick={handleNotebookLaunch}>Launch Notebook</Menu.Item>}
                  <Menu.Item onClick={handleCpuNotebookLaunch}>Launch CPU-only Notebook</Menu.Item>
                </Menu>
              )} placement="bottomRight"
              trigger={[ 'click' ]}
              onVisibleChange={handleVisibleChange}>
              <Button className={css.launchIcon}>
                <Icon name={isShowingCpu ? 'arrow-up': 'arrow-down'} size="tiny" />
              </Button>
            </Dropdown>
          </div>
        </section>
        <section className={css.top}>
          <NavigationItem icon="user" label="Dashboard" path="/det/dashboard" />
          <NavigationItem icon="experiment" label="Experiments" path="/det/experiments" />
          <NavigationItem icon="tasks" label="Tasks" path="/det/tasks" />
          <NavigationItem icon="cluster" label="Cluster" path="/det/cluster" />
        </section>
        <section className={css.bottom}>
          <NavigationItem icon="logs" label="Master Logs" path="/det/logs" popout />
          <NavigationItem icon="docs" label="Docs" path="/docs" popout />
          <NavigationItem icon="cloud" label="API" path="/swagger-ui" popout />
          <NavigationItem icon="collapse" label="Collapse" onClick={handleCollapse} />
        </section>
      </main>
      <footer>
        <Dropdown
          arrow
          overlay={(
            <Menu>
              <Menu.Item>
                <Link path={'/det/logout'}>Sign Out</Link>
              </Menu.Item>
            </Menu>
          )}
          placement="topLeft"
          trigger={[ 'click' ]}>
          <a className={css.user} href="#">
            <Avatar hideTooltip name={username} />
            {!ui.collapseChrome && <span>{username}</span>}
          </a>
        </Dropdown>
      </footer>
    </nav>
  ) : null;
};

export default Navigation;
