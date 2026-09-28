-- redeem-fns.sql — the §6 redeem() drafts for the race spike.
--
--   redeem_verbatim  the handoff §6 text, character-for-character semantics.
--   redeem_v2        the corrected draft: adds an explicit 'not_found' reason.
--                    §9/§19 name `not_found` in the arbitration taxonomy, but the
--                    verbatim function's reason subquery returns NULL for a code id
--                    that has no row — the spike pins that gap and proves v2 closes
--                    it without changing the race behavior.
--   naive_redeem     the check-then-act ANTI-pattern with a deliberately widened
--                    TOCTOU window (pg_sleep between read and write). Exists so the
--                    race test can be shown to DETECT the defect class — the
--                    red-first analog for a card that creates new files (roadmap
--                    "Greenfield note").
--
-- ⚠ SPIKE FIXTURE — applied only to the throwaway spike-supabase substrate.

create or replace function public.redeem_verbatim(p_code uuid, p_device text)
returns table (ok boolean, reason text) language plpgsql as $$
begin
  update public.codes set redeemed_by = p_device, redeemed_at = now()
   where id = p_code and redeemed_by is null and expires_at > now();
  if found then
    return query select true, null::text;
  else
    return query select false,
      (select case when redeemed_by is not null then 'already_used'
                   else 'expired' end
         from public.codes where id = p_code);
  end if;
end $$;

create or replace function public.redeem_v2(p_code uuid, p_device text)
returns table (ok boolean, reason text) language plpgsql as $$
begin
  update public.codes set redeemed_by = p_device, redeemed_at = now()
   where id = p_code and redeemed_by is null and expires_at > now();
  if found then
    return query select true, null::text;
  else
    return query select false, coalesce(
      (select case when c.redeemed_by is not null then 'already_used'
                   when c.expires_at <= now()     then 'expired'
              end
         from public.codes c where c.id = p_code),
      'not_found');
  end if;
end $$;

create or replace function public.naive_redeem(p_code uuid, p_device text)
returns table (ok boolean, reason text) language plpgsql as $$
declare v_free boolean;
begin
  select (redeemed_by is null and expires_at > now()) into v_free
    from public.codes where id = p_code;
  perform pg_sleep(0.4);  -- the TOCTOU window, widened so two callers reliably overlap
  if coalesce(v_free, false) then
    update public.codes set redeemed_by = p_device, redeemed_at = now()
     where id = p_code;
    return query select true, null::text;
  else
    return query select false, 'unavailable'::text;
  end if;
end $$;
