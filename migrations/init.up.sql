create schema if not exists review;

create table if not exists review.reviews (
    id bigserial primary key,
    product_id bigint not null,
    vendor_id bigint not null,
    author_user_id bigint not null,
    author_name text not null default '',
    rating integer not null check (rating between 1 and 5),
    comment text not null default '',
    excluded_from_rating boolean not null default false,
    deleted_at timestamp null,
    deleted_by_admin_id bigint null,
    created_at timestamp not null default now(),
    updated_at timestamp not null default now()
);

create unique index if not exists ux_reviews_product_author_active
    on review.reviews(product_id, author_user_id)
    where deleted_at is null;

create index if not exists ix_reviews_product_sort
    on review.reviews(product_id, deleted_at, created_at desc);

create index if not exists ix_reviews_vendor
    on review.reviews(vendor_id, deleted_at, created_at desc);

create table if not exists review.review_images (
    id bigserial primary key,
    review_id bigint not null references review.reviews(id) on delete cascade,
    url text not null,
    sort_order integer not null default 0
);

create table if not exists review.review_votes (
    review_id bigint not null references review.reviews(id) on delete cascade,
    user_id bigint not null,
    vote text not null check (vote in ('helpful', 'not_helpful')),
    created_at timestamp not null default now(),
    updated_at timestamp not null default now(),
    primary key (review_id, user_id)
);

create table if not exists review.review_reports (
    id bigserial primary key,
    review_id bigint not null references review.reviews(id) on delete cascade,
    reporter_user_id bigint not null,
    reason text not null,
    details text not null default '',
    status text not null default 'pending',
    created_at timestamp not null default now(),
    updated_at timestamp not null default now(),
    unique (review_id, reporter_user_id)
);

create table if not exists review.review_replies (
    id bigserial primary key,
    review_id bigint not null references review.reviews(id) on delete cascade,
    vendor_id bigint not null,
    comment text not null,
    deleted_at timestamp null,
    created_at timestamp not null default now(),
    updated_at timestamp not null default now(),
    unique (review_id)
);

create table if not exists review.review_disputes (
    id bigserial primary key,
    review_id bigint not null references review.reviews(id) on delete cascade,
    vendor_id bigint not null,
    reason text not null,
    status text not null default 'pending',
    admin_id bigint null,
    admin_comment text not null default '',
    created_at timestamp not null default now(),
    resolved_at timestamp null
);

create unique index if not exists ux_review_disputes_pending
    on review.review_disputes(review_id, vendor_id)
    where status = 'pending';

create index if not exists ix_review_disputes_status
    on review.review_disputes(status, created_at desc);
