FROM scratch

# Define arguments for building multi-arch images
ARG TARGETPLATFORM

# Copy artifacts to the container
COPY bin/${TARGETPLATFORM}/* /

# Expose ports
EXPOSE 80

# Provide default environment variables
ENV GIN_MODE="release"
ENV ENGINE="mongodb"
ENV PORT="80"
ENV ADDR="0.0.0.0"
ENV CHORE_INTERVAL="10s"
ENV CHORE_INITIAL_DELAY="10s"
ENV CHORE_INITIAL_RANDOM="true"
ENV PAGINATION_MAX_LIMIT="100"
ENV PAGINATION_MAX_OFFSET="10000"

# Specify entrypoint and default parameters
ENTRYPOINT ["/ratus"]
