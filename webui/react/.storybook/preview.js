import 'shared/styles/index.scss';
import 'shared/styles/storybook.scss';
import 'shared/prototypes';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';
import { ThemeOptions } from 'components/ThemeToggle';
import useTheme from 'hooks/useTheme';
import { BrowserRouter } from 'react-router-dom';
import { useEffect } from 'react';

export const globalTypes = {
  // https://storybook.js.org/addons/@storybook/addon-toolbars
  theme: {
    name: 'Theme',
    description: 'Global theme for components',
    defaultValue: ThemeOptions.light.displayName,
    toolbar: {
      icon: 'circlehollow',
      items: [
        ThemeOptions.light.displayName,
        ThemeOptions.dark.displayName,
        ThemeOptions.system.displayName,
      ],
      showName: true,
      dynamicTitle: true,
    },
  },
};

// ChildView is for calling useTheme in the top level of component
const ChildView = ({ context, children }) => {
  const storeDispatch = useStoreDispatch();
  useTheme();

  useEffect(() => {
    let currentTheme = ThemeOptions.light.className;

    switch (context.globals.theme) {
      case ThemeOptions.light.displayName:
        currentTheme = ThemeOptions.light.className;
        break;
      case ThemeOptions.dark.displayName:
        currentTheme = ThemeOptions.dark.className;
        break;
      case ThemeOptions.system.displayName:
        currentTheme = ThemeOptions.system.className;
        break;
      default:
        currentTheme = ThemeOptions.light.className;
    }
    storeDispatch({ type: StoreAction.SetMode, value: currentTheme });
  }, [context.globals.theme]);

  return <>{children}</>;
};

export const decorators = [
  (Story, context) => {
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
