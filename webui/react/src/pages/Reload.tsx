/*
 * The purpose of this page is to allow React to navigate from
 * an existing route to the same base route with different params.
 * For example, going from `/experiments/1` to `/experiments/2`.
 * Without the `Reload` redirection, `/experiments/2` will not
 * unmount and remount the page, causing stale data from experiment 1
 * to show up on experiment 2 page.
 */

import { useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';

const Reload: React.FC = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  useEffect(() => {
    if (searchParams.has('path')) {
      navigate(searchParams.get('path') as string, { replace: true });
    }
  }, [searchParams, navigate]);

  return null;
};

export default Reload;
