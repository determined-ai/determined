import useResize from './useResize';

const MOBILE_BREAKPOINT = 480;

const useMobile = (): boolean => {
  const { width } = useResize();

  return width < MOBILE_BREAKPOINT;
};

export default useMobile;
