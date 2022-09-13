import 'shared/styles/index.scss';
import 'shared/styles/storybook.scss';
import 'shared/prototypes';

import StoreProvider, { StoreAction, useStore, useStoreDispatch } from 'contexts/Store';
import { ThemeOptions } from 'components/ThemeToggle';
import useTheme from 'hooks/useTheme';
import { BrowserRouter } from 'react-router-dom';
import React, { useEffect } from 'react';
import { StoryContextForLoaders } from '@storybook/csf';
import { ReactFramework, Story } from '@storybook/react';
import { BrandingType } from 'types';

export const globalTypes = {
  // https://storybook.js.org/addons/@storybook/addon-toolbars
  theme: {
    name: 'Theme',
    description: 'Global theme for components',
    defaultValue: ThemeOptions.system.displayName,
    toolbar: {
      icon: 'circlehollow',
      items: [
        ThemeOptions.system.displayName,
        ThemeOptions.light.displayName,
        ThemeOptions.dark.displayName,
      ],
      showName: true,
      dynamicTitle: true,
    },
  },
};

// ChildView is for calling useTheme in the top level of component
const ChildView = ({ context, children }: {context: StoryContextForLoaders<ReactFramework>, children: React.ReactElement}) => {
  const storeDispatch = useStoreDispatch();
  const { ui } = useStore();
  console.log({ui});
  useTheme();

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
        version: ''
      } 
    });
  }, [])

  useEffect(() => {
    let currentTheme = ThemeOptions.system.className;
    console.log(context.globals)

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
    console.log(currentTheme)
    storeDispatch({ type: StoreAction.SetMode, value: currentTheme });
  }, [context.globals.theme]);

  return <>{children}</>;
};

export const decorators = [
  (Story: Story, context: StoryContextForLoaders<ReactFramework, typeof Story>) => {
    return (
      <StoreProvider>
        <BrowserRouter>
          <ChildView context={context}>
            <Story />
          </ChildView>
        </BrowserRouter>
      </StoreProvider>
    );
  },
];
export const parameters = { layout: 'centered' };
