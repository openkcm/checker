FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY checker .
USER 65532:65532

ENTRYPOINT ["checker"]
CMD [""]
