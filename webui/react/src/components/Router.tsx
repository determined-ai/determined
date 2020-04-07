import React from 'react';

import Route from 'components/Route';
import { RouteConfigItem } from 'routes';

interface Props {
  routes: RouteConfigItem[];
}

const Router: React.FC<Props> = (props: Props) => {
  return (
    <React.Fragment>
      {props.routes
        .filter(route => route.component != null)
        .map(route => <Route key={route.id} {...route} />)
      }
    </React.Fragment>
  );
};

export default Router;
