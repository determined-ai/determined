import useResize from './useResize';

const MOBILE_BREAKPOINT = 480;
const TABLET_BREAKPOINT = 1024;

const useMobile = (): boolean => {
  const { width } = useResize();

  return width < MOBILE_BREAKPOINT;
};
export const useTablet = (): boolean => {
  const { width } = useResize();

  return width <= TABLET_BREAKPOINT;
};

export default useMobile;
