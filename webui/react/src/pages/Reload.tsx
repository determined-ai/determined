/*
 * The purpose of this page is to allow React to navigate from
 * an existing route to the same base route with different params.
 * For example, going from `/experiments/1` to `/experiments/2`.
 * Without the `Reload` redirection, `/experiments/2` will not
 * unmount and remount the page, causing stale data from experiment 1
 * to show up on experiment 2 page.
 */

import queryString from 'query-string';
import { useEffect } from 'react';
import { useHistory, useLocation } from 'react-router-dom';

const Reload: React.FC = () => {
  const history = useHistory();
  const location = useLocation();

  useEffect(() => {
    const queryParams = queryString.parse(location.search);
    if (queryParams.path) {
      history.replace(queryParams.path as string);
    }
  }, [ history, location.search ]);

  return null;
};

export default Reload;
