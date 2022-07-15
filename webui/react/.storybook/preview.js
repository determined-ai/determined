import 'shared/styles/index.scss';
import 'shared/styles/storybook.scss';
import 'shared/prototypes';

import StoreProvider from 'contexts/Store';
import useTheme from 'hooks/useTheme';
import { BrowserRouter } from 'react-router-dom';

// ChildView is for calling useTheme in the top level of component
const ChildView = (props) => {
  useTheme();

  return <>{props.children}</>;
};

export const decorators = [
  (story) => {
    return (
      <StoreProvider>
        <BrowserRouter>
          <ChildView>{story()}</ChildView>
        </BrowserRouter>
      </StoreProvider>
    );
  },
];
export const parameters = { layout: 'centered' };
