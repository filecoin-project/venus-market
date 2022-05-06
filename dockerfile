FROM filvenus/venus-buildenv AS buildenv

RUN git clone https://github.com/filecoin-project/venus-market.git --depth 1 
RUN export GOPROXY=https://goproxy.cn && cd venus-market  && make deps && make
RUN cd venus-market && ldd ./venus-market


FROM filvenus/venus-runtime

# DIR for app
WORKDIR /app

# copy the app from build env
COPY --from=buildenv  /go/venus-market/venus-market /app/venus-market
COPY ./docker/script  /script

# copy ddl
COPY --from=buildenv  /usr/lib/x86_64-linux-gnu/libhwloc.so.5  \
                        /usr/lib/x86_64-linux-gnu/libOpenCL.so.1  \
                        /lib/x86_64-linux-gnu/libgcc_s.so.1  \
                        /lib/x86_64-linux-gnu/libutil.so.1  \
                        /lib/x86_64-linux-gnu/librt.so.1  \
                        /lib/x86_64-linux-gnu/libpthread.so.0  \
                        /lib/x86_64-linux-gnu/libm.so.6  \
                        /lib/x86_64-linux-gnu/libdl.so.2  \
                        /lib/x86_64-linux-gnu/libc.so.6  \
                        /usr/lib/x86_64-linux-gnu/libnuma.so.1  \
                        /usr/lib/x86_64-linux-gnu/libltdl.so.7  \
                        /lib/

EXPOSE 41235
ENTRYPOINT ["/app/venus-market"]


