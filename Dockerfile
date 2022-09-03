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
ENV MONGODB_URI="mongodb://mongo:27017"
ENV MONGODB_DATABASE="ratus"
ENV MONGODB_COLLECTION="tasks"
ENV MONGODB_RETENTION_PERIOD="72h"
ENV MONGODB_DISABLE_INDEX_CREATION="false"
ENV MONGODB_DISABLE_AUTO_FALLBACK="false"
ENV MONGODB_DISABLE_ATOMIC_POLL="false"

# Specify entrypoint and default parameters
ENTRYPOINT ["/ratus"]
