import 'shared/styles/index.scss';
import 'shared/styles/storybook.scss';
import 'shared/styles/antd.scss';
import 'shared/prototypes';

import { INITIAL_VIEWPORTS } from '@storybook/addon-viewport';
import { StoryContextForLoaders } from '@storybook/csf';
import { ReactFramework, Story } from '@storybook/react';
import React, { useEffect } from 'react';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { BrowserRouter } from 'react-router-dom';

import { ThemeOptions } from '../src/components/ThemeToggle';
import StoreProvider, { StoreAction, useStoreDispatch } from '../src/contexts/Store';
import useTheme from '../src/hooks/useTheme';
import useUI from '../src/shared/contexts/stores/UI';
import { BrandingType } from '../src/types';

export const globalTypes = {
  // https://storybook.js.org/addons/@storybook/addon-toolbars
  theme: {
    defaultValue: ThemeOptions.system.displayName,
    description: 'Global theme for components',
    name: 'Theme',
    toolbar: {
      dynamicTitle: true,
      icon: 'circlehollow',
      items: [
        ThemeOptions.system.displayName,
        ThemeOptions.light.displayName,
        ThemeOptions.dark.displayName,
      ],
      showName: true,
    },
  },
};

// ChildView is for calling useTheme in the top level of component
const ChildView = (
  { context, children }: {
    children: React.ReactElement,
    context: StoryContextForLoaders<ReactFramework>,
  },
) => {
  const storeDispatch = useStoreDispatch();
  useTheme();
  const { actions: uiActions } = useUI();

  useEffect(() => {
    // Have to set info.branding for useTheme to work
    storeDispatch({
      type: StoreAction.SetInfo,
      value: {
        branding: BrandingType.Determined,
        checked: false,
        clusterId: '',
        clusterName: 'storybook',
        isTelemetryEnabled: false,
        masterId: '',
        version: '',
      },
    });
    // Setting up a user in the store
    storeDispatch({
      type: StoreAction.SetUsers,
      value: [ {
        id: 1,
        isActive: true,
        isAdmin: true,
        username: 'Admin',
      } ],
    });
    storeDispatch({
      type: StoreAction.SetCurrentUser,
      value: {
        id: 1,
        isActive: true,
        isAdmin: true,
        username: 'Admin',
      },
    });
  }, [ storeDispatch ]);

  useEffect(() => {
    let currentTheme = ThemeOptions.system.className;

    switch (context.globals.theme) {
      case ThemeOptions.system.displayName:
        currentTheme = ThemeOptions.system.className;
        break;
      case ThemeOptions.light.displayName:
        currentTheme = ThemeOptions.light.className;
        break;
      case ThemeOptions.dark.displayName:
        currentTheme = ThemeOptions.dark.className;
        break;
      default:
        currentTheme = ThemeOptions.system.className;
    }
    uiActions.setMode(currentTheme);
  }, [ context.globals.theme, uiActions]);

  return <>{children}</>;
};

export const decorators = [
  (
    Story: Story,
    context: StoryContextForLoaders<ReactFramework, typeof Story>,
  ): React.ReactElement => {
    return (
      <StoreProvider>
        <BrowserRouter>
          <DndProvider backend={HTML5Backend}>
            <ChildView context={context}>
              <Story />
            </ChildView>
          </DndProvider>
        </BrowserRouter>
      </StoreProvider>
    );
  },
];
export const parameters = {
  layout: 'centered',
  options: {
    storySort: {
      method: 'alphabetical',
      order: [ 'Ant Design', 'Shared', 'Determined' ],
    },
  },
  viewport: { viewports: INITIAL_VIEWPORTS },
};
